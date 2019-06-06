// Package state contains the interface TransitionBuffer and its
// implementation
//
// Note: TransitionBuffer is not thread safe
//
// `Dequeue` will return the most relevant `Proposed` transition for the given
// `Height`.
package state

import (
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
)

// A Transition is an event that transitions a `Machine` from one
// State to another. It is generated externally to the `Machine`.
type Transition interface {
	IsTransition()
	Round() block.Round
	Signer() sig.Signatory
}

// An external event has triggered a Tick.
type Ticked struct {
	time.Time
}

// IsTransition implements the `Transition` interface for the
// `Ticked` event.
func (Ticked) IsTransition() {
}

// Signer implements the `Transition` interface for the
// `Ticked` event.
func (Ticked) Signer() sig.Signatory {
	return sig.Signatory{}
}

// Round implements the `Transition` interface for the
// `Ticked` event.
func (Ticked) Round() block.Round {
	return -1
}

// A Proposed block has been received by another Replica.
type Proposed struct {
	block.SignedPropose
}

// IsTransition implements the `Transition` interface for the
// `Proposed` event.
func (Proposed) IsTransition() {
}

// Round implements the `Transition` interface for the
// `Proposed` event.
func (proposed Proposed) Round() block.Round {
	return proposed.SignedPropose.Round
}

// Signer implements the `Transition` interface for the
// `Proposed` event.
func (proposed Proposed) Signer() sig.Signatory {
	return proposed.SignedPropose.Signatory
}

// A PreVoted block has been signed and broadcast by another
// `Replica`.
type PreVoted struct {
	block.SignedPreVote
}

// IsTransition implements the `Transition` interface for the
// `PreVoted` event.
func (PreVoted) IsTransition() {
}

// Round implements the `Transition` interface for the
// `PreVoted` event.
func (prevoted PreVoted) Round() block.Round {
	return prevoted.SignedPreVote.Round
}

// Signer implements the `Transition` interface for the
// `PreVoted` event.
func (prevoted PreVoted) Signer() sig.Signatory {
	return prevoted.Signatory
}

// A PreCommitted polka has been signed and broadcast by another
// `Replica`.
type PreCommitted struct {
	block.SignedPreCommit
}

// IsTransition implements the `Transition` interface for the
// `PreCommitted` event.
func (PreCommitted) IsTransition() {
}

// Round implements the `Transition` interface for the
// `PreCommitted` event.
func (precommitted PreCommitted) Round() block.Round {
	return precommitted.SignedPreCommit.Polka.Round
}

// Signer implements the `Transition` interface for the
// `PreCommitted` event.
func (precommitted PreCommitted) Signer() sig.Signatory {
	return precommitted.Signatory
}

// A TransitionBuffer is used to temporarily buffer `Proposed` that
// are not ready to be processed because they are from higher rounds. All
// `Transitions` are buffered against their respective `Height` and
// will be dequeued one by one.
type TransitionBuffer interface {
	Enqueue(transition Transition)
	Dequeue(height block.Height) (Transition, bool)
	// Drop everything below the given Height. You should call this
	// the moment you know everything below the current height is
	// meaningless.
	Drop(height block.Height)
}

type transitionBuffer struct {
	queues map[block.Height][]Proposed
	cap    int
}

// NewTransitionBuffer creates an empty TransitionBuffer with a maximum queue capacity.
func NewTransitionBuffer(cap int) TransitionBuffer {
	return &transitionBuffer{
		queues: make(map[block.Height][]Proposed),
		cap:    cap,
	}
}

func (buffer *transitionBuffer) Enqueue(transition Transition) {
	switch transition := transition.(type) {
	case Proposed:
		buffer.newQueue(transition.Block.Height)
		buffer.queues[transition.Block.Height] = append(buffer.queues[transition.Block.Height], transition)
	default:
		panic(fmt.Sprintf("unsupported transition %T", transition))
	}
}

// Dequeue takes the next Proposed for the provided height. If there is
// nothing at that height or the queue is empty it will return false.
func (buffer *transitionBuffer) Dequeue(height block.Height) (Transition, bool) {
	if queue, ok := buffer.queues[height]; ok && len(queue) > 0 {
		transition := queue[0]
		buffer.queues[height] = queue[1:]
		return transition, true
	}
	return nil, false
}

// Drop deletes all entries below the provided height. I assume you
// will call `Drop` with your current `Height` whenever you are done
// processing all previous `Height`s to prevent `TransitionBuffer`
// from becoming a memory leak.
func (buffer *transitionBuffer) Drop(height block.Height) {
	for k := range buffer.queues {
		if k < height {
			delete(buffer.queues, k)
		}
	}
}

func (buffer *transitionBuffer) newQueue(height block.Height) {
	if _, ok := buffer.queues[height]; !ok {
		buffer.queues[height] = make([]Proposed, 0, buffer.cap)
	}
}
