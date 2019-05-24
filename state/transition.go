// Package state contains the interface TransitionBuffer and its
// implementation
//
// Note: TransitionBuffer is not thread safe
//
// There are two types of `Transition`, those with a `Height` and those
// that are "immediate". Any "immediate" `Transition`s will be
// `Dequeue`ed first, regardless of the provided `Height`. Otherwise,
// `Dequeue` will return the most relevant `Transition` for the given
// `Height`. For example: you will not get a `PreVoted` if a
// `PreCommitted` was already `Enqueue`ed at that `Height`.
//
// Keep in mind`Transition`s that don't have a `Height` are not pruned.
// For example: if you `Enqueue` a `TimedOut` twice then the next two
// `Dequeue` will return a `TimedOut`. The "immediate" `Transition`s are
// stored in a FIFO queue.
package state

import (
	"time"

	"github.com/renproject/hyperdrive/block"
)

// A Transition is an event that transitions a `Machine` from one
// State to another. It is generated externally to the `Machine`.
type Transition interface {
	IsTransition()
}

// TimedOut waiting for some other external event.
// FIXME: TimedOut should probably have Height and Round
type TimedOut struct {
	time.Time
}

// IsTransition implements the `Transition` interface for the
// `TimedOut` event.
func (TimedOut) IsTransition() {
}

// An external event has triggered a Tick.
type Ticked struct {
	time.Time
}

// IsTransition implements the `Transition` interface for the
// `Ticked` event.
func (Ticked) IsTransition() {
}

// A Proposed block has been received by another Replica.
type Proposed struct {
	block.SignedPropose
}

// IsTransition implements the `Transition` interface for the
// `Proposed` event.
func (Proposed) IsTransition() {
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

// A PreCommitted polka has been signed and broadcast by another
// `Replica`.
type PreCommitted struct {
	block.SignedPreCommit
}

// IsTransition implements the `Transition` interface for the
// `PreCommitted` event.
func (PreCommitted) IsTransition() {
}

// A TransitionBuffer is used to temporarily buffer `Transitions` that
// are not ready to be processed because of the `State`. All
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
	queues map[block.Height][]Transition
	cap    int
}

// NewTransitionBuffer creates an empty TransitionBuffer with a maximum queue capacity.
func NewTransitionBuffer(cap int) TransitionBuffer {
	return &transitionBuffer{
		queues: make(map[block.Height][]Transition),
		cap:    cap,
	}
}

func (buffer *transitionBuffer) Enqueue(transition Transition) {
	switch transition := transition.(type) {
	case Proposed:
		buffer.newQueue(transition.Block.Height)
		if len(buffer.queues[transition.Block.Height]) > 0 {
			switch buffer.queues[transition.Block.Height][0].(type) {
			case PreVoted, PreCommitted:
			default:
				buffer.queues[transition.Block.Height] = append(buffer.queues[transition.Block.Height], transition)
			}
		} else {
			buffer.queues[transition.Block.Height] = append(buffer.queues[transition.Block.Height], transition)
		}

	case PreVoted:
		buffer.newQueue(transition.Height)
		if len(buffer.queues[transition.Height]) > 0 {
			switch buffer.queues[transition.Height][0].(type) {
			case Proposed:
				buffer.queues[transition.Height] = []Transition{transition}
			case PreCommitted:
			default:
				buffer.queues[transition.Height] = append(buffer.queues[transition.Height], transition)
			}
		} else {
			buffer.queues[transition.Height] = append(buffer.queues[transition.Height], transition)
		}

	case PreCommitted:
		buffer.newQueue(transition.Polka.Height)
		if len(buffer.queues[transition.Polka.Height]) > 0 {
			switch buffer.queues[transition.Polka.Height][0].(type) {
			case Proposed:
				buffer.queues[transition.Polka.Height] = []Transition{}
			case PreVoted:
				buffer.queues[transition.Polka.Height] = []Transition{}
			default:
			}
		}
		buffer.queues[transition.Polka.Height] = append(buffer.queues[transition.Polka.Height], transition)

	default:
	}
}

// Dequeue picks things that don't have a height first, like timeouts
// then takes the next Transition for the provided height. If there is
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
		buffer.queues[height] = make([]Transition, 0, buffer.cap)
	}
}
