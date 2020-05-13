package process

import (
	"bytes"
	"fmt"
	"io"

	"github.com/renproject/id"
	"github.com/renproject/surge"
)

// The State of a Process. It should be saved after every method call on the
// Process, but should not be saved during method calls (interacting with the
// State concurently is unsafe). It is worth noting that the State does not
// contain a decision array, because it delegates this responsibility to the
// Committer interface.
//
// L1:
//
//  Initialization:
//      currentHeight := 0 /* current height, or consensus instance we are currently executing */
//      currentRound  := 0 /* current round number */
//      currentStep ∈ {propose, prevote, precommit}
//      decision[]  := nil
//      lockedValue := nil
//      lockedRound := −1
//      validValue  := nil
//      validRound  := −1
type State struct {
	CurrentHeight Height `json:"currentHeight"`
	CurrentRound  Round  `json:"currentRound"`
	CurrentStep   Step   `json:"currentStep"`
	LockedValue   Value  `json:"lockedValue"` // The most recent value for which a precommit message has been sent.
	LockedRound   Round  `json:"lockedRound"` // The last round in which the process sent a precommit message that is not nil.
	ValidValue    Value  `json:"validValue"`  // The most recent possible decision value.
	ValidRound    Round  `json:"validRound"`  // The last round in which valid value is updated.
}

// DefaultState returns a State with all fields set to their default values. The
// Height default to 1, because the genesis block is assumed to exist at Height
// 0.
func DefaultState() State {
	return State{
		CurrentHeight: 1, // Skip genesis.
		CurrentRound:  0,
		CurrentStep:   Proposing,
		LockedValue:   NilValue,
		LockedRound:   InvalidRound,
		ValidValue:    NilValue,
		ValidRound:    InvalidRound,
	}
}

// Equal compares two States. If they are equal, then it returns true, otherwise
// it returns false.
func (state State) Equal(other *State) bool {
	return state.CurrentHeight == other.CurrentHeight &&
		state.CurrentRound == other.CurrentRound &&
		state.CurrentStep == other.CurrentStep &&
		state.LockedValue.Equal(&other.LockedValue) &&
		state.LockedRound == other.LockedRound &&
		state.ValidValue.Equal(&other.ValidValue) &&
		state.ValidRound == other.ValidRound
}

func (state State) SizeHint() int {
	return surge.SizeHint(state.CurrentHeight) +
		surge.SizeHint(state.CurrentRound) +
		surge.SizeHint(state.CurrentStep) +
		surge.SizeHint(state.LockedValue) +
		surge.SizeHint(state.LockedRound) +
		surge.SizeHint(state.ValidValue) +
		surge.SizeHint(state.ValidRound)
}

func (state State) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, state.CurrentHeight, m)
	if err != nil {
		return m, fmt.Errorf("marshaling current height=%v: %v", state.CurrentHeight, err)
	}
	m, err = surge.Marshal(w, state.CurrentRound, m)
	if err != nil {
		return m, fmt.Errorf("marshaling current round=%v: %v", state.CurrentRound, err)
	}
	m, err = surge.Marshal(w, state.CurrentStep, m)
	if err != nil {
		return m, fmt.Errorf("marshaling current step=%v: %v", state.CurrentStep, err)
	}
	m, err = surge.Marshal(w, state.LockedValue, m)
	if err != nil {
		return m, fmt.Errorf("marshaling locked value=%v: %v", state.LockedValue, err)
	}
	m, err = surge.Marshal(w, state.LockedRound, m)
	if err != nil {
		return m, fmt.Errorf("marshaling locked round=%v: %v", state.LockedRound, err)
	}
	m, err = surge.Marshal(w, state.ValidValue, m)
	if err != nil {
		return m, fmt.Errorf("marshaling valid value=%v: %v", state.ValidValue, err)
	}
	m, err = surge.Marshal(w, state.ValidRound, m)
	if err != nil {
		return m, fmt.Errorf("marshaling valid round=%v: %v", state.ValidRound, err)
	}
	return m, nil
}

func (state *State) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &state.CurrentHeight, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling current height: %v", err)
	}
	m, err = surge.Unmarshal(r, &state.CurrentRound, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling current round: %v", err)
	}
	m, err = surge.Unmarshal(r, &state.CurrentStep, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling current step: %v", err)
	}
	m, err = surge.Unmarshal(r, &state.LockedValue, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling locked value: %v", err)
	}
	m, err = surge.Unmarshal(r, &state.LockedRound, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling locked round: %v", err)
	}
	m, err = surge.Unmarshal(r, &state.ValidValue, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling valid value: %v", err)
	}
	m, err = surge.Unmarshal(r, &state.ValidRound, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling valid round: %v", err)
	}
	return m, nil
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

// Pid defines a type alias for hashes that represent the unique identity of a
// Process in the consensus algorithm. No distrinct Processes should ever have
// the same Pid, and a Process must maintain the same Pid for its entire life.
type Pid = id.Signatory

// Pids defines a typedef for a slice of Pids.
type Pids []Pid

// Equal compares two slices of Pids. If they are equal, the it returns true,
// otherwise it returns false.
func (pids Pids) Equal(other Pids) bool {
	if len(pids) != len(other) {
		return false
	}
	for i := range pids {
		if !pids[i].Equal(&other[i]) {
			return false
		}
	}
	return true
}

// Contains checks for the existence of a Pid in the slice of Pids. If the Pid
// is in the slice, then it returns true, otherwise it returns false. The
// complexity of this method is O(n), so it is recommended that the results are
// cached whenever the Pids slice is large.
func (pids Pids) Contains(pid Pid) bool {
	for i := range pids {
		if pids[i].Equal(&pid) {
			return true
		}
	}
	return false
}

// Set returns the slice of Pids, converted into a PidSet. This is convenient
// for checking the existence of Pids when ordering does not matter.
func (pids Pids) Set() PidSet {
	set := PidSet{}
	for _, pid := range pids {
		set[pid] = true
	}
	return set
}

// PidSet defines a typedef for a map of Pids.
type PidSet map[Pid]bool
