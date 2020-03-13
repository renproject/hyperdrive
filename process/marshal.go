package process

import (
	"fmt"
	"io"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
	"github.com/renproject/surge"
)

func (propose Propose) SizeHint() int {
	return surge.SizeHint(propose.sig) +
		surge.SizeHint(propose.signatory) +
		surge.SizeHint(propose.height) +
		surge.SizeHint(propose.round) +
		surge.SizeHint(propose.block) +
		surge.SizeHint(propose.validRound) +
		surge.SizeHint(propose.latestCommit.Block) +
		surge.SizeHint(propose.latestCommit.Precommits) +
		surge.SizeHint(propose.signatory)
}

func (propose Propose) Marshal(w io.Writer, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	m, err := surge.Marshal(w, propose.sig, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.round, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.block, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.validRound, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.latestCommit.Block, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.latestCommit.Precommits, m); err != nil {
		return m, err
	}
	return surge.Marshal(w, propose.signatory, m)
}

func (propose *Propose) Unmarshal(r io.Reader, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	m, err := surge.Unmarshal(r, &propose.sig, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.round, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.block, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.validRound, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.latestCommit.Block, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.latestCommit.Precommits, m); err != nil {
		return m, err
	}
	return surge.Unmarshal(r, &propose.signatory, m)
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// `Propose` type.
func (propose Propose) MarshalBinary() ([]byte, error) {
	return surge.ToBinary(propose)
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// `Propose` type.
func (propose *Propose) UnmarshalBinary(data []byte) error {
	return surge.FromBinary(data, propose)
}

func (prevote Prevote) SizeHint() int {
	return surge.SizeHint(prevote.sig) +
		surge.SizeHint(prevote.signatory) +
		surge.SizeHint(prevote.height) +
		surge.SizeHint(prevote.round) +
		surge.SizeHint(prevote.blockHash) +
		surge.SizeHint(prevote.nilReasons)
}

func (prevote Prevote) Marshal(w io.Writer, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	m, err := surge.Marshal(w, prevote.sig, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, prevote.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, prevote.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, prevote.round, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, prevote.blockHash, m); err != nil {
		return m, err
	}
	return surge.Marshal(w, prevote.nilReasons, m)
}

func (prevote *Prevote) Unmarshal(r io.Reader, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	m, err := surge.Unmarshal(r, &prevote.sig, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &prevote.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &prevote.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &prevote.round, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &prevote.blockHash, m); err != nil {
		return m, err
	}
	return surge.Unmarshal(r, &prevote.nilReasons, m)
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// `Prevote` type.
func (prevote Prevote) MarshalBinary() ([]byte, error) {
	return surge.ToBinary(prevote)
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// `Prevote` type.
func (prevote *Prevote) UnmarshalBinary(data []byte) error {
	return surge.FromBinary(data, prevote)
}

func (precommit Precommit) SizeHint() int {
	return surge.SizeHint(precommit.sig) +
		surge.SizeHint(precommit.signatory) +
		surge.SizeHint(precommit.height) +
		surge.SizeHint(precommit.round) +
		surge.SizeHint(precommit.blockHash)
}

func (precommit Precommit) Marshal(w io.Writer, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	m, err := surge.Marshal(w, precommit.sig, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, precommit.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, precommit.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, precommit.round, m); err != nil {
		return m, err
	}
	return surge.Marshal(w, precommit.blockHash, m)
}

func (precommit *Precommit) Unmarshal(r io.Reader, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	m, err := surge.Unmarshal(r, &precommit.sig, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &precommit.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &precommit.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &precommit.round, m); err != nil {
		return m, err
	}
	return surge.Unmarshal(r, &precommit.blockHash, m)
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// `Precommit` type.
func (precommit Precommit) MarshalBinary() ([]byte, error) {
	return surge.ToBinary(precommit)
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// `Precommit` type.
func (precommit *Precommit) UnmarshalBinary(data []byte) error {
	return surge.FromBinary(data, precommit)
}

func (inbox Inbox) SizeHint() int {
	return surge.SizeHint(uint32(inbox.f)) + surge.SizeHint(inbox.messages)
}

func (inbox Inbox) Marshal(w io.Writer, m int) (int, error) {
	// Write f.
	m, err := surge.Marshal(w, int64(inbox.f), m)
	if err != nil {
		return m, err
	}
	// Write message type.
	if m, err = surge.Marshal(w, inbox.messageType, m); err != nil {
		return m, err
	}

	// Write the number of heights.
	if m, err = surge.Marshal(w, uint32(len(inbox.messages)), m); err != nil {
		return m, err
	}

	// Write rounds for each height.
	for height, rounds := range inbox.messages {
		// Write the height.
		if m, err = surge.Marshal(w, height, m); err != nil {
			return m, err
		}
		{
			// Write the number of rounds at this height.
			if m, err = surge.Marshal(w, uint32(len(rounds)), m); err != nil {
				return m, err
			}
			// Write middle map.
			for round, signatories := range rounds {
				// Write middle key.
				if m, err := surge.Marshal(w, round, m); err != nil {
					return m, err
				}
				// Write middle value.
				{
					// Write middle map length.
					if m, err = surge.Marshal(w, uint32(len(signatories)), m); err != nil {
						return m, err
					}
					// Write middle map.
					for signatory, message := range signatories {
						if m, err = surge.Marshal(w, signatory, m); err != nil {
							return m, err
						}
						if m, err = surge.Marshal(w, message, m); err != nil {
							return m, err
						}
					}
				}
			}
		}
	}
	return m, nil
}

func (inbox *Inbox) Unmarshal(r io.Reader, m int) (int, error) {
	// Read f.
	var f int64
	m, err := surge.Unmarshal(r, &f, m)
	if err != nil {
		return m, fmt.Errorf("cannot unmarshal f: %v", err)
	}
	inbox.f = int(f)
	// Read message type.
	if m, err = surge.Unmarshal(r, &inbox.messageType, m); err != nil {
		return m, err
	}

	// Read the number of heights.
	numHeights := uint32(0)
	if m, err = surge.Unmarshal(r, &numHeights, m); err != nil {
		return m, fmt.Errorf("cannot unmarshal number of heights: %v", err)
	}
	if int(numHeights) < 0 {
		return m, fmt.Errorf("cannot unmarshal number of heights: unexpected negative length")
	}
	m -= int(numHeights)
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}
	// Read rounds for each height.
	inbox.messages = map[block.Height]map[block.Round]map[id.Signatory]Message{}
	for j := uint32(0); j < numHeights; j++ {
		// Read the height.
		height := block.Height(0)
		if m, err = surge.Unmarshal(r, &height, m); err != nil {
			return m, fmt.Errorf("cannot unmarshal height: %v", err)
		}
		// Read the number of rounds at this height.
		numRounds := uint32(0)
		if m, err = surge.Unmarshal(r, &numRounds, m); err != nil {
			return m, fmt.Errorf("cannot unmarshal number of rounds: %v", err)
		}
		if int(numRounds) < 0 {
			return m, fmt.Errorf("cannot unmarshal number of rounds: unexpected negative length")
		}
		m -= int(numRounds)
		if m <= 0 {
			return m, surge.ErrMaxBytesExceeded
		}

		inbox.messages[height] = map[block.Round]map[id.Signatory]Message{}
		for i2 := uint32(0); i2 < numRounds; i2++ {
			// Read middle key.
			k2 := block.Round(0)
			if m, err = surge.Unmarshal(r, &k2, m); err != nil {
				return m, fmt.Errorf("cannot unmarshal middle key: %v", err)
			}
			// Read middle map length.
			l3 := uint32(0)
			if m, err = surge.Unmarshal(r, &l3, m); err != nil {
				return m, fmt.Errorf("cannot unmarshal middle length: %v", err)
			}
			if int(l3) < 0 {
				return m, fmt.Errorf("expected negative length")
			}
			m -= int(l3)
			if m <= 0 {
				return m, surge.ErrMaxBytesExceeded
			}
			inbox.messages[height][k2] = map[id.Signatory]Message{}
			for i3 := uint32(0); i3 < l3; i3++ {
				// Read inner key.
				k3 := id.Signatory{}
				if m, err = surge.Unmarshal(r, &k3, m); err != nil {
					return m, fmt.Errorf("cannot unmarshal inner key: %v", err)
				}
				// Read inner value.
				switch inbox.messageType {
				case ProposeMessageType:
					message := new(Propose)
					if m, err = message.Unmarshal(r, m); err != nil {
						return m, fmt.Errorf("cannot unmarshal propose: %v", err)
					}
					inbox.messages[height][k2][k3] = message
				case PrevoteMessageType:
					message := new(Prevote)
					if m, err = message.Unmarshal(r, m); err != nil {
						return m, fmt.Errorf("cannot unmarshal prevote: %v", err)
					}
					inbox.messages[height][k2][k3] = message
				case PrecommitMessageType:
					message := new(Precommit)
					if m, err = message.Unmarshal(r, m); err != nil {
						return m, fmt.Errorf("cannot unmarshal precommit: %v", err)
					}
					inbox.messages[height][k2][k3] = message
				default:
					return m, fmt.Errorf("unsupported MessageType=%v", inbox.messageType)
				}
			}
		}
	}
	return m, nil
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// `Inbox` type.
func (inbox Inbox) MarshalBinary() ([]byte, error) {
	return surge.ToBinary(inbox)
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// `Inbox` type. See the `UnmarshalJSON` method for more information.
func (inbox *Inbox) UnmarshalBinary(data []byte) error {
	return surge.FromBinary(data, inbox)
}

func (state State) SizeHint() int {
	return surge.SizeHint(state.CurrentHeight) +
		surge.SizeHint(state.CurrentRound) +
		surge.SizeHint(state.CurrentStep) +
		surge.SizeHint(state.LockedBlock) +
		surge.SizeHint(state.LockedRound) +
		surge.SizeHint(state.ValidBlock) +
		surge.SizeHint(state.ValidRound) +
		surge.SizeHint(state.Proposals) +
		surge.SizeHint(state.Prevotes) +
		surge.SizeHint(state.Precommits)
}

func (state State) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, state.CurrentHeight, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, state.CurrentRound, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, state.CurrentStep, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, state.LockedBlock, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, state.LockedRound, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, state.ValidBlock, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, state.ValidRound, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, state.Proposals, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, state.Prevotes, m); err != nil {
		return m, err
	}
	return surge.Marshal(w, state.Precommits, m)
}

func (state *State) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &state.CurrentHeight, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &state.CurrentRound, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &state.CurrentStep, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &state.LockedBlock, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &state.LockedRound, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &state.ValidBlock, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &state.ValidRound, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, state.Proposals, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, state.Prevotes, m); err != nil {
		return m, err
	}
	return surge.Unmarshal(r, state.Precommits, m)
}
