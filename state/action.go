package state

import "github.com/renproject/hyperdrive/block"

// An Action is emitted by the state Machine to signal to other packages that
// some Action needs to be broadcast to other state Machines that are
// participating in the consensus algorithm.
type Action interface {

	// IsAction is a marker function. It is implemented by types to ensure that
	// we cannot accidentally use the wrong types is some functions. We use type
	// switching is used to enumerate the possible concrete types.
	IsAction()
}

// Propose a Block for consensus in the current round. A previously found Commit
// can be included to help locked state Machines to unlock.
type Propose struct {
	block.SignedPropose

	LastCommit Commit
}

// IsAction is a marker function that implements the Action interface for the Propose type.
func (Propose) IsAction() {
}

type PreVote struct {
	block.PreVote
}

// IsAction is a marker function that implements the Action interface for the PreVote type.
func (PreVote) IsAction() {
}

type SignedPreVote struct {
	block.SignedPreVote
}

// IsAction is a marker function that implements the Action interface for the SignedPreVote type.
func (SignedPreVote) IsAction() {
}

type PreCommit struct {
	block.PreCommit
}

// IsAction is a marker function that implements the Action interface for the PreCommit type.
func (PreCommit) IsAction() {
}

type SignedPreCommit struct {
	block.SignedPreCommit
}

// IsAction is a marker function that implements the Action interface for the SignedPreCommit type.
func (SignedPreCommit) IsAction() {
}

type Commit struct {
	block.Commit
}

// IsAction is a marker function that implements the Action interface for the Commit type.
func (Commit) IsAction() {
}
