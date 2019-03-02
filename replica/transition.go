package replica

import (
	"time"

	"github.com/renproject/hyperdrive/block"
)

// A Transition is an event that transitions a `StateMachine` from one State to another. It is generated externally to
// the `StateMachine`.
type Transition interface {
	IsTransition()
}

// TimedOut waiting for some other external event.
type TimedOut struct {
	time.Time
}

// IsTransition implements the `Transition` interface for the `TimedOut` event.
func (timedOut TimedOut) IsTransition() {
}

// A Proposed block has been received by another Replica.
type Proposed struct {
	block.Block
}

// IsTransition implements the `Transition` interface for the `Proposed` event.
func (proposed Proposed) IsTransition() {
}

// A PreVoted block has been signed and broadcast by another `Replica`.
type PreVoted struct {
	block.SignedPreVote
}

// IsTransition implements the `Transition` interface for the `PreVoted` event.
func (preVoted PreVoted) IsTransition() {
}

// A PreCommitted polka has been signed and broadcast by another `Replica`.
type PreCommitted struct {
	block.SignedPreCommit
}

// IsTransition implements the `Transition` interface for the `PreCommitted` event.
func (preCommitted PreCommitted) IsTransition() {
}

type Action interface {
	IsAction()
}

type PreVote struct {
	block.PreVote
}

func (preVote PreVote) IsAction() {
}

type PreCommit struct {
	block.PreCommit
}

func (preCommit PreCommit) IsAction() {
}
