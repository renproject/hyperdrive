// Package consensus contains the interface TransitionBuffer and its
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
package consensus

import (
	"time"

	"github.com/renproject/hyperdrive/v1/block"
)

// A Transition is an event that transitions a `StateMachine` from one
// State to another. It is generated externally to the `StateMachine`.
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
func (timedOut TimedOut) IsTransition() {
}

// A Proposed block has been received by another Replica.
type Proposed struct {
	block.SignedBlock
}

// IsTransition implements the `Transition` interface for the
// `Proposed` event.
func (proposed Proposed) IsTransition() {
}

// A PreVoted block has been signed and broadcast by another
// `Replica`.
type PreVoted struct {
	block.SignedPreVote
}

// IsTransition implements the `Transition` interface for the
// `PreVoted` event.
func (preVoted PreVoted) IsTransition() {
}

// A PreCommitted polka has been signed and broadcast by another
// `Replica`.
type PreCommitted struct {
	block.SignedPreCommit
}

// IsTransition implements the `Transition` interface for the
// `PreCommitted` event.
func (preCommitted PreCommitted) IsTransition() {
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
	queues map[block.Height]*transitionQueue
	cap    int
}

// NewTransitionBuffer creates an empty TransitionBuffer with a maximum queue capacity.
func NewTransitionBuffer(cap int) TransitionBuffer {
	return &transitionBuffer{
		queues: make(map[block.Height]*transitionQueue),
		cap:    cap,
	}
}

func (buffer *transitionBuffer) Enqueue(transition Transition) {
	switch transition := transition.(type) {
	case Proposed:
		buffer.newQueue(transition.Height)
		queue := buffer.queues[transition.Height]
		if tran, ok := queue.peek(); ok {
			switch tran.(type) {
			case PreVoted:
				// Don't enqueue
			case PreCommitted:
				// Don't enqueue
			default:
				queue.enqueue(transition)
			}
		} else {
			queue.enqueue(transition)
		}
	case PreVoted:
		buffer.newQueue(transition.Height)
		queue := buffer.queues[transition.Height]
		if tran, ok := queue.peek(); ok {
			switch tran.(type) {
			case Proposed:
				queue.reset()
				queue.enqueue(transition)
			case PreCommitted:
				// Don't enqueue
			default:
				queue.enqueue(transition)
			}
		} else {
			queue.enqueue(transition)
		}
	case PreCommitted:
		buffer.newQueue(transition.Polka.Height)
		queue := buffer.queues[transition.Polka.Height]
		if tran, ok := queue.peek(); ok {
			switch tran.(type) {
			case Proposed:
				queue.reset()
			case PreVoted:
				queue.reset()
			case PreCommitted:
			default:
			}
		}
		queue.enqueue(transition)
	default:
		// Ignore the Transition and do not buffer it
	}
}

// Dequeue picks things that don't have a height first, like timeouts
// then takes the next Transition for the provided height. If there is
// nothing at that height or the queue is empty it will return false.
func (buffer *transitionBuffer) Dequeue(height block.Height) (Transition, bool) {
	if queue, ok := buffer.queues[height]; ok {
		return queue.dequeue()
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
		buffer.queues[height] = &transitionQueue{
			queue: make([]Transition, buffer.cap),
			end:   0,
		}
	}
}

type transitionQueue struct {
	queue []Transition
	end   int
}

func (tq *transitionQueue) enqueue(tran Transition) {
	if len(tq.queue) == tq.end {
		tq.queue = append(tq.queue, tran)
	} else {
		tq.queue[tq.end] = tran
	}
	tq.end++
}

func (tq *transitionQueue) dequeue() (Transition, bool) {
	if tq.end == 0 {
		return nil, false
	}
	tq.end--
	return tq.queue[tq.end], true
}

func (tq *transitionQueue) reset() {
	tq.end = 0
}

func (tq *transitionQueue) peek() (Transition, bool) {
	if tq.end == 0 {
		return nil, false
	}
	return tq.queue[tq.end-1], true
}
