package process

import (
	"github.com/renproject/hyperdrive/process/block"
	"github.com/renproject/hyperdrive/process/message"
)

// The State of a Process. It is isolated from the Process so that it can be
// easily marshaled to/from JSON.
type State struct {
	CurrentHeight block.Height `json:"currentHeight"`
	CurrentRound  block.Round  `json:"currentRound"`
	CurrentStep   Step         `json:"currentStep"`

	LockedBlock block.Block `json:"lockedBlock"`
	LockedRound block.Round `json:"lockedRound"`
	ValidBlock  block.Block `json:"validBlock"`
	ValidRound  block.Round `json:"validRound"`

	Proposals  message.Inbox `json:"proposals"`
	Prevotes   message.Inbox `json:"prevotes"`
	Precommits message.Inbox `json:"precommits"`
}

// DefaultState returns a State with all values set to the default. See
// https://arxiv.org/pdf/1807.04938.pdf for more information.
func DefaultState() State {
	return State{
		CurrentHeight: 0,
		CurrentRound:  0,
		CurrentStep:   StepPropose,

		LockedBlock: block.InvalidBlock,
		LockedRound: block.InvalidRound,
		ValidBlock:  block.InvalidBlock,
		ValidRound:  block.InvalidRound,
	}
}

// Reset the State (not all values are reset). See
// https://arxiv.org/pdf/1807.04938.pdf for more information.
func (state *State) Reset() {
	state.LockedBlock = block.InvalidBlock
	state.LockedRound = block.InvalidRound
	state.ValidBlock = block.InvalidBlock
	state.ValidRound = block.InvalidRound
}
