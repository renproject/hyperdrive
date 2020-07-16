package replica

import (
	"context"

	"github.com/renproject/hyperdrive/mq"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/scheduler"
	"github.com/renproject/hyperdrive/timer"
	"github.com/renproject/id"
)

// DidHandleMessage is called by the Replica after it has finished handling an
// input message (i.e. Propose, Prevote, or Precommit), or timeout.
type DidHandleMessage func()

// A Replica represents one Process in a replicated state machine that is bound
// to a specific Shard. It signs Messages before sending them to other Replicas,
// and verifies Messages before accepting them from other Replicas.
type Replica struct {
	opts Options

	proc         process.Process
	procsAllowed map[id.Signatory]bool

	onTimeoutPropose   <-chan timer.Timeout
	onTimeoutPrevote   <-chan timer.Timeout
	onTimeoutPrecommit <-chan timer.Timeout

	onPropose   chan process.Propose
	onPrevote   chan process.Prevote
	onPrecommit chan process.Precommit
	mq          mq.MessageQueue

	didHandleMessage DidHandleMessage
}

// New instantiates and returns a pointer to a new Hyperdrive replica machine
func New(
	opts Options,
	whoami id.Signatory,
	signatories []id.Signatory,
	propose process.Proposer,
	validate process.Validator,
	commit process.Committer,
	catch process.Catcher,
	broadcast process.Broadcaster,
	didHandleMessage DidHandleMessage,
) *Replica {
	f := len(signatories) / 3
	onTimeoutPropose := make(chan timer.Timeout, 10)
	onTimeoutPrevote := make(chan timer.Timeout, 10)
	onTimeoutPrecommit := make(chan timer.Timeout, 10)
	timer := timer.NewLinearTimer(opts.TimerOpts, onTimeoutPropose, onTimeoutPrevote, onTimeoutPrecommit)
	scheduler := scheduler.NewRoundRobin(signatories)
	proc := process.New(
		whoami,
		f,
		timer,
		scheduler,
		propose,
		validate,
		broadcast,
		commit,
		catch,
	)

	procsAllowed := make(map[id.Signatory]bool)
	for _, signatory := range signatories {
		procsAllowed[signatory] = true
	}

	return &Replica{
		opts: opts,

		proc:         proc,
		procsAllowed: procsAllowed,

		onTimeoutPropose:   onTimeoutPropose,
		onTimeoutPrevote:   onTimeoutPrevote,
		onTimeoutPrecommit: onTimeoutPrecommit,

		onPropose:   make(chan process.Propose, opts.MessageQueueOpts.MaxCapacity),
		onPrevote:   make(chan process.Prevote, opts.MessageQueueOpts.MaxCapacity),
		onPrecommit: make(chan process.Precommit, opts.MessageQueueOpts.MaxCapacity),
		mq:          mq.New(opts.MessageQueueOpts),

		didHandleMessage: didHandleMessage,
	}
}

// Run starts the Hyperdrive replica's process
func (replica *Replica) Run(ctx context.Context) {
	replica.proc.Start()

	isRunning := true
	for isRunning {
		func() {
			defer func() {
				if replica.didHandleMessage != nil {
					replica.didHandleMessage()
				}
			}()

			select {
			case <-ctx.Done():
				isRunning = false
				return

			case timeout := <-replica.onTimeoutPropose:
				replica.proc.OnTimeoutPropose(timeout.Height, timeout.Round)
			case timeout := <-replica.onTimeoutPrevote:
				replica.proc.OnTimeoutPrevote(timeout.Height, timeout.Round)
			case timeout := <-replica.onTimeoutPrecommit:
				replica.proc.OnTimeoutPrecommit(timeout.Height, timeout.Round)

			case propose := <-replica.onPropose:
				if !replica.filterHeight(propose.Height) {
					return
				}
				if !replica.filterFrom(propose.From) {
					return
				}
				replica.mq.InsertPropose(propose)
			case prevote := <-replica.onPrevote:
				if !replica.filterHeight(prevote.Height) {
					return
				}
				if !replica.filterFrom(prevote.From) {
					return
				}
				replica.mq.InsertPrevote(prevote)
			case precommit := <-replica.onPrecommit:
				if !replica.filterHeight(precommit.Height) {
					return
				}
				if !replica.filterFrom(precommit.From) {
					return
				}
				replica.mq.InsertPrecommit(precommit)
			}

			replica.flush()
		}()
	}
}

// Propose adds a propose message to the replica. This message will be
// asynchronously inserted into the replica's message queue asynchronously,
// and consumed when the replica does not have any immediate task to do
func (replica *Replica) Propose(ctx context.Context, propose process.Propose) {
	select {
	case <-ctx.Done():
	case replica.onPropose <- propose:
	}
}

// Prevote adds a prevote message to the replica. This message will be
// asynchronously inserted into the replica's message queue asynchronously,
// and consumed when the replica does not have any immediate task to do
func (replica *Replica) Prevote(ctx context.Context, prevote process.Prevote) {
	select {
	case <-ctx.Done():
	case replica.onPrevote <- prevote:
	}
}

// Precommit adds a precommit message to the replica. This message will be
// asynchronously inserted into the replica's message queue asynchronously,
// and consumed when the replica does not have any immediate task to do
func (replica *Replica) Precommit(ctx context.Context, precommit process.Precommit) {
	select {
	case <-ctx.Done():
	case replica.onPrecommit <- precommit:
	}
}

func (replica *Replica) filterHeight(height process.Height) bool {
	return height >= replica.proc.CurrentHeight
}

func (replica *Replica) filterFrom(from id.Signatory) bool {
	return replica.procsAllowed[from]
}

func (replica *Replica) flush() {
	for {
		n := replica.mq.Consume(
			replica.proc.CurrentHeight,
			replica.proc.Propose,
			replica.proc.Prevote,
			replica.proc.Precommit,
		)
		if n == 0 {
			return
		}
	}
}
