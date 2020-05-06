package proc

import (
	"bytes"

	"github.com/renproject/id"
)

// The State of a Process. It is isolated from the Process so that it can be
// easily marshaled to/from JSON.
type State struct {
	CurrentHeight Height `json:"currentHeight"`
	CurrentRound  Round  `json:"currentRound"`
	CurrentStep   Step   `json:"currentStep"`
	LockedValue   Value  `json:"lockedValue"` // The most recent value for which a precommit message has been sent.
	LockedRound   Round  `json:"lockedRound"` // The last round in which the process sent a precommit message that is not nil.
	ValidValue    Value  `json:"validValue"`  // The most recent possible decision value.
	ValidRound    Round  `json:"validRound"`  // The last round in which valid value is updated.
}

// DefaultState returns a State with all values set to their default. See
// https://arxiv.org/pdf/1807.04938.pdf for more information.
func DefaultState() State {
	return State{
		CurrentHeight: 1, // Skip genesis.
		CurrentRound:  0,
		CurrentStep:   Proposing,
		LockedValue:   Value{},
		LockedRound:   InvalidRound,
		ValidValue:    Value{},
		ValidRound:    InvalidRound,
	}
}

// Reset the State (not all values are reset). See
// https://arxiv.org/pdf/1807.04938.pdf for more information.
func (state *State) Reset() {
	state.LockedValue = Value{}
	state.LockedRound = InvalidRound

	state.ValidValue = Value{}
	state.ValidRound = InvalidRound
}

// Equal compares one State with another. If they are equal, then it returns
// true, otherwise it returns false.
func (state *State) Equal(other *State) bool {
	return state.CurrentHeight == other.CurrentHeight &&
		state.CurrentRound == other.CurrentRound &&
		state.CurrentStep == other.CurrentStep &&
		state.LockedValue.Equal(&other.LockedValue) &&
		state.LockedRound == other.LockedRound &&
		state.ValidValue.Equal(&other.ValidValue) &&
		state.ValidRound == other.ValidRound
}

// Step defines a typedef for uint8 values that represent the step of the state
// of a Process partaking in the consensus algorithm.
type Step uint8

// Enumerate step values.
const (
	Proposing     = Step(0)
	Prevoting     = Step(1)
	Precommitting = Step(2)
)

// Height defines a typedef for int64 values that represent the height of a
// Value at which the consensus algorithm is attempting to reach consensus.
type Height int64

// Round defines a typedef for int64 values that represent the round of a Value
// at which the consensus algorithm is attempting to reach consensus.
type Round int64

const (
	// InvalidRound is a reserved int64 that represents an invalid Round. It is
	// used when a Process is trying to represent that it does have have a
	// LockedRound or ValidRound.
	InvalidRound = Round(-1)
)

// Value defines a typedef for hashes that represent the hashes of proposed
// values in the consensus algorithm. In the context of a blockchain, a Value
// would be a block.
type Value id.Hash

// Equal compares two Values. If they are equal, then it returns true, otherwise
// it returns false.
func (v *Value) Equal(other *Value) bool {
	return bytes.Equal(v[:], other[:])
}

var (
	// NilValue is a reserved hash that represents when a Process is
	// prevoting/precommitting to nothing (i.e. the Process wants to progress to
	// the next Round).
	NilValue = Value(id.Hash{})
)

// Pid defines a typedef for hsahes that represent the unique identity of a
// Process in the consensus algorithm. No distrinct Processes should ever have
// the same Pid, and a Process must maintain the same Pid for its entire life.
type Pid id.Hash

// Equal compares two Pids. If they are equal, the it returns true, otherwise it
// returns false.
func (pid *Pid) Equal(other *Pid) bool {
	return bytes.Equal(pid[:], other[:])
}
