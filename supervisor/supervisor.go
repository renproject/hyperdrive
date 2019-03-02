package supervisor

import (
	"errors"
	"fmt"
	"sync"
)

// ErrSupervisorStopped is returned when reading restart errors from a `Supervisor` that has been stopped.
var ErrSupervisorStopped = errors.New("supervisor stopped")

// A Context is passed to a `Runner` when it is spawned into a `Supervisor`. It allows the `Supervisor` to signal to the
// `Runner` that it needs to terminate, and allows the `Runner` to signal to the `Supervisor` that it wants to
// terminate.
type Context interface {

	// Done is closed when `Runners` attached to the `Context` should terminate.
	Done() <-chan struct{}

	// Kill signals to the `Supervisor` that at least one `Runner` attached to the `Context` wants to terminate.
	Kill(error)

	// Error returns the error that killed the `Context`.
	Error() error
}

type context struct {
	doneMu     *sync.Mutex
	doneClosed bool
	doneErr    error
	done       chan struct{}
}

func newContext() Context {
	return &context{
		doneMu:     new(sync.Mutex),
		doneClosed: false,
		doneErr:    nil,
		done:       make(chan struct{}),
	}
}

// Done returns a read-only channel that is closed when the `Context` is killed.
func (ctx *context) Done() <-chan struct{} {
	return ctx.done
}

// Kill the `Context` by closing the channel returned when calling `Done`. Only the first call to `Kill` will have an
// effect.
func (ctx *context) Kill(err error) {
	ctx.doneMu.Lock()
	defer ctx.doneMu.Unlock()

	if ctx.doneClosed {
		return
	}
	ctx.doneClosed = true
	ctx.doneErr = err
	close(ctx.done)
}

// Error returns the error that killed the `Context`. It returns `nil` when the `Context` is still alive.
func (ctx *context) Error() error {
	ctx.doneMu.Lock()
	defer ctx.doneMu.Unlock()

	return ctx.doneErr
}

// A Runner is spawned into a `Supervisor` that will
type Runner interface {
	Run(ctx Context)
}

// RunnerFunc is a wrapper type around a function that accepts a `Context` as its only argument. It allows the function
// to be used as a `Runner`.
type RunnerFunc func(ctx Context)

// Run implementst the `Runner` interface. It calls the wrapped function and passes it the `Context`.
func (runner RunnerFunc) Run(ctx Context) {
	runner(ctx)
}

// RestartPolicy defines what the `Supervisor` will do when one of its `Runners` has terminated.
type RestartPolicy int

const (
	// RestartNone will cause the `Supervisor` to kill all `Runners` and not restart any of them.
	RestartNone RestartPolicy = 0
	// RestartAll will cause the `Supervisor` to kill all `Runners` and restart all of them by calling the `Run` method
	// again.
	RestartAll RestartPolicy = 1
)

// A Supervisor spawns multiple `Runners` and monitors them. If a `Runner` terminates, then the `Supervisor` will kill
// all other `Runners` and use its `RestartPolicy` to recover.
type Supervisor interface {
	Spawn(runners ...Runner)
	SpawnFunc(runners ...func(ctx Context))
	Start()
	Stop()
	Restarts() <-chan error
}

type supervisor struct {
	wg sync.WaitGroup

	ctxMu *sync.Mutex
	ctx   Context

	runningMu       *sync.Mutex
	runningQuit     chan struct{}
	runningQuitting chan struct{}
	running         bool

	runnersMu *sync.Mutex
	runners   []Runner

	restartPolicy RestartPolicy
	restarts      chan error
}

// New returns a `Supervisor` that can be user to spawn `Runners`. The `RestartPolicy` will be used for all `Runners`.
func New(restartPolicy RestartPolicy) Supervisor {
	switch restartPolicy {
	case RestartNone, RestartAll:
	default:
		panic(fmt.Errorf("unexpected `RestartPolicy`: %v", restartPolicy))
	}

	return &supervisor{
		ctxMu: new(sync.Mutex),
		ctx:   newContext(),

		runningMu:       new(sync.Mutex),
		runningQuit:     make(chan struct{}),
		runningQuitting: make(chan struct{}),
		running:         false,

		runnersMu: new(sync.Mutex),
		runners:   make([]Runner, 0),

		restartPolicy: restartPolicy,
		restarts:      make(chan error, 1024),
	}
}

// Spawn a `Runner` into the `Supervisor`. All `Runners` that are spawned after the `Supervisor` has started running
// will be ignored.
func (sv *supervisor) Spawn(runners ...Runner) {
	for _, runner := range runners {
		sv.spawn(runner)
	}
}

// SpawnFunc into the `Supervisor`. The function will be wrapped by the `RunnerFunc` type. All `Runners` that are
// spawned after the `Supervisor` has started running will be ignored.
func (sv *supervisor) SpawnFunc(runners ...func(ctx Context)) {
	for _, runner := range runners {
		sv.spawn(RunnerFunc(runner))
	}
}

// Start the `Supervisor`. All `Runners` that have been spawned into the `Supervisor` will begin running. All `Runners`
// that are spawned after the `Supervisor` has started will be ignored.
func (sv *supervisor) Start() {
	if !sv.start() {
		return
	}

	defer sv.quit()

	for {
		sv.run()
		if !sv.wait() {
			return
		}
		if !sv.restart() {
			return
		}
	}
}

// Stop the `Supervisor` and terminate all `Runners`.
func (sv *supervisor) Stop() {

	select {
	case <-sv.runningQuitting:
	default:
		close(sv.runningQuitting)
	}

	sv.runningMu.Lock()
	running := sv.running
	sv.runningMu.Unlock()

	if !running {
		return
	}

	sv.ctxMu.Lock()
	sv.ctx.Kill(nil)
	sv.ctxMu.Unlock()

	<-sv.runningQuit
}

// Restarts returns a read-only channel of errors thrown by`Runners`. The channel has a non-zero capacity but the
// `Supervisor` will not block when writing. The channel will be closed when the `Supervisor` is stopped.
func (sv *supervisor) Restarts() <-chan error {
	return sv.restarts
}

func (sv *supervisor) spawn(runner Runner) {
	sv.runningMu.Lock()
	defer sv.runningMu.Unlock()

	if sv.running {
		return
	}

	sv.runnersMu.Lock()
	defer sv.runnersMu.Unlock()

	sv.runners = append(sv.runners, runner)
}

func (sv *supervisor) spawnAll() {
	sv.runnersMu.Lock()
	defer sv.runnersMu.Unlock()

	for _, runner := range sv.runners {
		sv.wg.Add(1)
		go func(runner Runner) {
			defer sv.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					sv.ctxMu.Lock()
					defer sv.ctxMu.Unlock()

					sv.ctx.Kill(fmt.Errorf("runner crashed: %v", r))
				}
			}()

			runner.Run(sv.ctx)
		}(runner)
	}
}

func (sv *supervisor) start() bool {
	sv.runningMu.Lock()
	defer sv.runningMu.Unlock()

	if sv.running {
		return false
	}
	sv.running = true
	sv.runningQuit = make(chan struct{})

	return true
}

func (sv *supervisor) restart() bool {
	sv.runningMu.Lock()
	defer sv.runningMu.Unlock()

	sv.running = true
	sv.runningQuit = make(chan struct{})

	return true
}

func (sv *supervisor) run() {
	sv.ctxMu.Lock()
	defer sv.ctxMu.Unlock()

	sv.ctx = newContext()
	sv.spawnAll()
}

func (sv *supervisor) wait() bool {

	<-sv.ctx.Done()

	err := sv.ctx.Error()
	if err != nil {
		select {
		case sv.restarts <- err:
		default:
		}
	}

	sv.wg.Wait()

	select {
	case <-sv.runningQuitting:
		return false
	default:
	}

	switch sv.restartPolicy {
	case RestartNone:
		return false
	case RestartAll:
		return true
	}
	return false
}

func (sv *supervisor) quit() {
	sv.runningMu.Lock()
	defer sv.runningMu.Unlock()

	sv.running = false

	select {
	case sv.restarts <- ErrSupervisorStopped:
	default:
	}

	select {
	case <-sv.runningQuit:
	default:
		close(sv.runningQuit)
	}
}
