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
// input message (i.e. Propose, Prevote, or Precommit), or timeout. The message
// could have been either accepted and inserted into the processing queue, or
// filtered out and dropped. The callback is also called when the context
// within which the Replica runs gets cancelled.
type DidHandleMessage func()

// A Replica represents a process in a replicated state machine that
// participates in the Hyperdrive Consensus Algorithm. It encapsulates a
// Hyperdrive Process and exposes an interface for the Hyperdrive user to
// insert messages (propose, prevote, precommit, timeouts). A Replica then
// handles these messages asynchronously, after sorting them in an increasing
// order of height and round. A Replica is instantiated by passing in the set
// of signatories participating in the consensus mechanism, and it filters out
// messages that have not been sent by one of the known set of allowed
// signatories.
type Replica struct {
	opts Options

	proc         process.Process
	procsAllowed map[id.Signatory]bool

	mch chan interface{}
	mq  mq.MessageQueue

	didHandleMessage DidHandleMessage
}

// New instantiates and returns a pointer to a new Hyperdrive replica machine
func New(
	opts Options,
	whoami id.Signatory,
	signatories []id.Signatory,
	linearTimer process.Timer,
	propose process.Proposer,
	validate process.Validator,
	commit process.Committer,
	catch process.Catcher,
	broadcast process.Broadcaster,
	didHandleMessage DidHandleMessage,
) *Replica {
	f := len(signatories) / 3
	scheduler := scheduler.NewRoundRobin(signatories)
	proc := process.New(
		whoami,
		f,
		linearTimer,
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

		mch: make(chan interface{}, opts.MessageQueueOpts.MaxCapacity),
		mq:  mq.New(opts.MessageQueueOpts),

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
			case m := <-replica.mch:
				switch m := m.(type) {
				case timer.Timeout:
					switch m.MessageType {
					case process.MessageTypePropose:
						replica.proc.OnTimeoutPropose(m.Height, m.Round)
					case process.MessageTypePrevote:
						replica.proc.OnTimeoutPrevote(m.Height, m.Round)
					case process.MessageTypePrecommit:
						replica.proc.OnTimeoutPrecommit(m.Height, m.Round)
					default:
						return
					}
				case process.Propose:
					if !replica.filterHeight(m.Height) {
						return
					}
					if !replica.filterFrom(m.From) {
						return
					}
					replica.mq.InsertPropose(m)
				case process.Prevote:
					if !replica.filterHeight(m.Height) {
						return
					}
					if !replica.filterFrom(m.From) {
						return
					}
					replica.mq.InsertPrevote(m)
				case process.Precommit:
					if !replica.filterHeight(m.Height) {
						return
					}
					if !replica.filterFrom(m.From) {
						return
					}
					replica.mq.InsertPrecommit(m)
				}
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
	case replica.mch <- propose:
	}
}

// Prevote adds a prevote message to the replica. This message will be
// asynchronously inserted into the replica's message queue asynchronously,
// and consumed when the replica does not have any immediate task to do
func (replica *Replica) Prevote(ctx context.Context, prevote process.Prevote) {
	select {
	case <-ctx.Done():
	case replica.mch <- prevote:
	}
}

// Precommit adds a precommit message to the replica. This message will be
// asynchronously inserted into the replica's message queue asynchronously,
// and consumed when the replica does not have any immediate task to do
func (replica *Replica) Precommit(ctx context.Context, precommit process.Precommit) {
	select {
	case <-ctx.Done():
	case replica.mch <- precommit:
	}
}

// TimeoutPropose adds a propose timeout message to the replica. This message
// will be filtered based on the replica's consensus height, and inserted
// asynchronously into the replica's message queue. It will be consumed when
// the replica does not have any immediate task to do
func (replica *Replica) TimeoutPropose(ctx context.Context, timeout timer.Timeout) {
	select {
	case <-ctx.Done():
	case replica.mch <- timeout:
	}
}

// TimeoutPrevote adds a prevote timeout message to the replica. This message
// will be filtered based on the replica's consensus height, and inserted
// asynchronously into the replica's message queue. It will be consumed when
// the replica does not have any immediate task to do
func (replica *Replica) TimeoutPrevote(ctx context.Context, timeout timer.Timeout) {
	select {
	case <-ctx.Done():
	case replica.mch <- timeout:
	}
}

// TimeoutPrecommit adds a precommit timeout message to the replica. This message
// will be filtered based on the replica's consensus height, and inserted
// asynchronously into the replica's message queue. It will be consumed when
// the replica does not have any immediate task to do
func (replica *Replica) TimeoutPrecommit(ctx context.Context, timeout timer.Timeout) {
	select {
	case <-ctx.Done():
	case replica.mch <- timeout:
	}
}

// ResetHeight of the underlying process to a future height. This is should only
// be used when resynchronising the chain. If the given height is less than or
// equal to the current height, nothing happens.
func (replica *Replica) ResetHeight(newHeight process.Height) {
	if newHeight <= replica.proc.State.CurrentHeight {
		return
	}
	replica.proc.State = process.DefaultState().WithCurrentHeight(newHeight)
}

// State returns the current height, round and step of the underlying process.
func (replica Replica) State() (process.Height, process.Round, process.Step) {
	return replica.proc.CurrentHeight, replica.proc.CurrentRound, replica.proc.CurrentStep
}

// CurrentHeight returns the current height of the underlying process.
func (replica Replica) CurrentHeight() process.Height {
	return replica.proc.CurrentHeight
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
