package process

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/renproject/hyperdrive/block"
)

// The State of a Process. It is isolated from the Process so that it can be
// easily marshaled to/from JSON.
type State struct {
	CurrentHeight block.Height `json:"currentHeight"`
	CurrentRound  block.Round  `json:"currentRound"`
	CurrentStep   Step         `json:"currentStep"`

	LockedBlock block.Block `json:"lockedBlock"` // the most recent block for which a PRECOMMIT message has been sent
	LockedRound block.Round `json:"lockedRound"` // the last round in which the process sent a PRECOMMIT message that is not nil.
	ValidBlock  block.Block `json:"validBlock"`  // store the most recent possible decision value
	ValidRound  block.Round `json:"validRound"`  // is the last round in which valid value is updated

	Proposals  *Inbox `json:"proposals"`
	Prevotes   *Inbox `json:"prevotes"`
	Precommits *Inbox `json:"precommits"`
}

// DefaultState returns a State with all values set to the default. See
// https://arxiv.org/pdf/1807.04938.pdf for more information.
func DefaultState(f int) State {
	return State{
		CurrentHeight: 1, // Skip the genesis block
		CurrentRound:  0,
		CurrentStep:   StepPropose,
		LockedBlock:   block.InvalidBlock,
		LockedRound:   block.InvalidRound,
		ValidBlock:    block.InvalidBlock,
		ValidRound:    block.InvalidRound,
		Proposals:     NewInbox(f, ProposeMessageType),
		Prevotes:      NewInbox(f, PrevoteMessageType),
		Precommits:    NewInbox(f, PrecommitMessageType),
	}
}

// Reset the State (not all values are reset). See
// https://arxiv.org/pdf/1807.04938.pdf for more information.
func (state *State) Reset(height block.Height) {
	state.LockedBlock = block.InvalidBlock
	state.LockedRound = block.InvalidRound
	state.ValidBlock = block.InvalidBlock
	state.ValidRound = block.InvalidRound
	state.Proposals.Reset(height)
	state.Prevotes.Reset(height)
	state.Precommits.Reset(height)
}

// Equal compares one State with another.
func (state *State) Equal(other State) bool {
	return state.CurrentHeight == other.CurrentHeight &&
		state.CurrentRound == other.CurrentRound &&
		state.CurrentStep == other.CurrentStep &&
		state.LockedBlock.Equal(other.LockedBlock) &&
		state.LockedRound == other.LockedRound &&
		state.ValidBlock.Equal(other.ValidBlock) &&
		state.ValidRound == other.ValidRound
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// State type.
func (state State) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, state.CurrentHeight); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.CurrentHeight: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, state.CurrentRound); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.CurrentRound: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, state.CurrentStep); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.CurrentStep: %v", err)
	}
	lockedBlockData, err := state.LockedBlock.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal state.LockedBlock: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(lockedBlockData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.LockedBlock len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, lockedBlockData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.LockedBlock data: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, state.LockedRound); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.LockedRound: %v", err)
	}
	validBlockData, err := state.ValidBlock.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal state.ValidBlock: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(validBlockData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.ValidBlock len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, validBlockData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.ValidBlock data: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, state.ValidRound); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.ValidRound: %v", err)
	}
	proposalsData, err := state.Proposals.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal state.Proposals: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(proposalsData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.Proposals len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, proposalsData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.Proposals data: %v", err)
	}
	prevotesData, err := state.Prevotes.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal state.Prevotes: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(prevotesData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.Prevotes len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, prevotesData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.Prevotes data: %v", err)
	}
	precommitsData, err := state.Precommits.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal state.Precommits: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(precommitsData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.Precommits len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, precommitsData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write state.Precommits data: %v", err)
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// State type.
func (state *State) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.LittleEndian, &state.CurrentHeight); err != nil {
		return fmt.Errorf("cannot read state.CurrentHeight: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &state.CurrentRound); err != nil {
		return fmt.Errorf("cannot read state.CurrentRound: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &state.CurrentStep); err != nil {
		return fmt.Errorf("cannot read state.CurrentStep: %v", err)
	}
	var numBytes uint64
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read state.LockedBlock len: %v", err)
	}
	lockedBlockBytes := make([]byte, numBytes)
	if _, err := buf.Read(lockedBlockBytes); err != nil {
		return fmt.Errorf("cannot read state.LockedBlock data: %v", err)
	}
	if err := state.LockedBlock.UnmarshalBinary(lockedBlockBytes); err != nil {
		return fmt.Errorf("cannot unmarshal state.LockedBlock: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &state.LockedRound); err != nil {
		return fmt.Errorf("cannot read state.LockedRound: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read state.ValidBlock len: %v", err)
	}
	validBlockBytes := make([]byte, numBytes)
	if _, err := buf.Read(validBlockBytes); err != nil {
		return fmt.Errorf("cannot read state.ValidBlock data: %v", err)
	}
	if err := state.ValidBlock.UnmarshalBinary(validBlockBytes); err != nil {
		return fmt.Errorf("cannot unmarshal state.ValidBlock: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &state.ValidRound); err != nil {
		return fmt.Errorf("cannot read state.ValidRound: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read state.Proposals len: %v", err)
	}
	proposalsBytes := make([]byte, numBytes)
	if _, err := buf.Read(proposalsBytes); err != nil {
		return fmt.Errorf("cannot read state.Proposals data: %v", err)
	}
	if err := state.Proposals.UnmarshalBinary(proposalsBytes); err != nil {
		return fmt.Errorf("cannot unmarshal state.Proposals: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read state.Prevotes len: %v", err)
	}
	prevotesBytes := make([]byte, numBytes)
	if _, err := buf.Read(prevotesBytes); err != nil {
		return fmt.Errorf("cannot read state.Prevotes data: %v", err)
	}
	if err := state.Prevotes.UnmarshalBinary(prevotesBytes); err != nil {
		return fmt.Errorf("cannot unmarshal state.Prevotes: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read state.Precommits len: %v", err)
	}
	precommitsBytes := make([]byte, numBytes)
	if _, err := buf.Read(precommitsBytes); err != nil {
		return fmt.Errorf("cannot read state.Precommits data: %v", err)
	}
	if err := state.Precommits.UnmarshalBinary(precommitsBytes); err != nil {
		return fmt.Errorf("cannot unmarshal state.Precommits: %v", err)
	}
	return nil
}
