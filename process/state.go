package process

import (
	"bytes"
	"fmt"

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

	// ProposeLogs store the Proposes for all Rounds.
	ProposeLogs map[Round]Propose `json:"proposeLogs"`
	// PrevoteLogs store the Prevotes for all Processes in all Rounds.
	PrevoteLogs map[Round]map[id.Signatory]Prevote `json:"prevoteLogs"`
	// PrecommitLogs store the Precommits for all Processes in all Rounds.
	PrecommitLogs map[Round]map[id.Signatory]Precommit `json:"precommitLogs"`
	// OnceFlags prevents events from happening more than once.
	OnceFlags map[Round]OnceFlag `json:"onceFlags"`
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

		ProposeLogs:   make(map[Round]Propose),
		PrevoteLogs:   make(map[Round]map[id.Signatory]Prevote),
		PrecommitLogs: make(map[Round]map[id.Signatory]Precommit),
		OnceFlags:     make(map[Round]OnceFlag),
	}
}

// Clone the State into another copy that can be modified without affecting the
// original.
func (state State) Clone() State {
	cloned := State{
		CurrentHeight: state.CurrentHeight,
		CurrentRound:  state.CurrentRound,
		CurrentStep:   state.CurrentStep,
		LockedValue:   state.LockedValue,
		LockedRound:   state.LockedRound,
		ValidValue:    state.ValidValue,
		ValidRound:    state.ValidRound,

		ProposeLogs:   make(map[Round]Propose),
		PrevoteLogs:   make(map[Round]map[id.Signatory]Prevote),
		PrecommitLogs: make(map[Round]map[id.Signatory]Precommit),
		OnceFlags:     make(map[Round]OnceFlag),
	}
	for round, propose := range state.ProposeLogs {
		cloned.ProposeLogs[round] = propose
	}
	for round, prevotes := range state.PrevoteLogs {
		cloned.PrevoteLogs[round] = make(map[id.Signatory]Prevote)
		for signatory, prevote := range prevotes {
			cloned.PrevoteLogs[round][signatory] = prevote
		}
	}
	for round, precommits := range state.PrecommitLogs {
		cloned.PrecommitLogs[round] = make(map[id.Signatory]Precommit)
		for signatory, precommit := range precommits {
			cloned.PrecommitLogs[round][signatory] = precommit
		}
	}
	for round, onceFlag := range state.OnceFlags {
		cloned.OnceFlags[round] = onceFlag
	}
	return cloned
}

// Equal compares two States. If they are equal, then it returns true, otherwise
// it returns false. Message logs and once-flags are ignored for the purpose of
// equality.
func (state State) Equal(other *State) bool {
	return state.CurrentHeight == other.CurrentHeight &&
		state.CurrentRound == other.CurrentRound &&
		state.CurrentStep == other.CurrentStep &&
		state.LockedValue.Equal(&other.LockedValue) &&
		state.LockedRound == other.LockedRound &&
		state.ValidValue.Equal(&other.ValidValue) &&
		state.ValidRound == other.ValidRound
}

// SizeHint implements the Surge SizeHinter interface, and returns the byte size
// of the state instance
func (state State) SizeHint() int {
	return surge.SizeHint(state.CurrentHeight) +
		surge.SizeHint(state.CurrentRound) +
		surge.SizeHint(state.CurrentStep) +
		surge.SizeHint(state.LockedValue) +
		surge.SizeHint(state.LockedRound) +
		surge.SizeHint(state.ValidValue) +
		surge.SizeHint(state.ValidRound) +
		surge.SizeHint(state.ProposeLogs) +
		surge.SizeHint(state.PrevoteLogs) +
		surge.SizeHint(state.PrecommitLogs) +
		surge.SizeHint(state.OnceFlags)
}

// Marshal implements the Surge Marshaler interface
func (state State) Marshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Marshal(state.CurrentHeight, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling current height=%v: %v", state.CurrentHeight, err)
	}
	buf, rem, err = surge.Marshal(state.CurrentRound, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling current round=%v: %v", state.CurrentRound, err)
	}
	buf, rem, err = surge.Marshal(state.CurrentStep, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling current step=%v: %v", state.CurrentStep, err)
	}
	buf, rem, err = surge.Marshal(state.LockedValue, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling locked value=%v: %v", state.LockedValue, err)
	}
	buf, rem, err = surge.Marshal(state.LockedRound, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling locked round=%v: %v", state.LockedRound, err)
	}
	buf, rem, err = surge.Marshal(state.ValidValue, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling valid value=%v: %v", state.ValidValue, err)
	}
	buf, rem, err = surge.Marshal(state.ValidRound, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling valid round=%v: %v", state.ValidRound, err)
	}
	buf, rem, err = surge.Marshal(state.ProposeLogs, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling %v propose logs: %v", len(state.ProposeLogs), err)
	}
	buf, rem, err = surge.Marshal(state.PrevoteLogs, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling %v prevote logs: %v", len(state.PrevoteLogs), err)
	}
	buf, rem, err = surge.Marshal(state.PrecommitLogs, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling %v precommit logs: %v", len(state.PrecommitLogs), err)
	}
	buf, rem, err = surge.Marshal(state.OnceFlags, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling %v once flags: %v", len(state.OnceFlags), err)
	}
	return buf, rem, nil
}

// Unmarshal implements the Surge Unmarshaler interface
func (state *State) Unmarshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Unmarshal(&state.CurrentHeight, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling current height: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.CurrentRound, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling current round: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.CurrentStep, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling current step: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.LockedValue, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling locked value: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.LockedRound, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling locked round: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.ValidValue, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling valid value: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.ValidRound, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling valid round: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.ProposeLogs, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling propose logs: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.PrevoteLogs, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling prevote logs: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.PrecommitLogs, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling precommit logs: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&state.OnceFlags, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling once flags: %v", err)
	}
	return buf, rem, nil
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

func (v Value) MarshalJSON() ([]byte, error) {
	return id.Hash(v).MarshalJSON()
}

func (v *Value) UnmarshalJSON(data []byte) error {
	return (*id.Hash)(v).UnmarshalJSON(data)
}

func (v Value) String() string {
	return id.Hash(v).String()
}

var (
	// NilValue is a reserved hash that represents when a Process is
	// prevoting/precommitting to nothing (i.e. the Process wants to progress to
	// the next Round).
	NilValue = Value(id.Hash{})
)
