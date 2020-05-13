package process

import (
	"fmt"
	"io"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
	"github.com/renproject/surge"
)

// SizeHint of how many bytes will be needed to represent this propose in
// binary.
func (propose Propose) SizeHint() int {
	return surge.SizeHint(propose.shard) +
		surge.SizeHint(propose.signatory) +
		surge.SizeHint(propose.sig) +
		surge.SizeHint(propose.height) +
		surge.SizeHint(propose.round) +
		surge.SizeHint(propose.block) +
		surge.SizeHint(propose.validRound) +
		surge.SizeHint(propose.latestCommit.Block) +
		surge.SizeHint(propose.latestCommit.Precommits)
}

// Marshal this propose into binary.
func (propose Propose) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, propose.shard, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, propose.sig, m); err != nil {
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
	return surge.Marshal(w, propose.latestCommit.Precommits, m)
}

// Unmarshal into this propose from binary.
func (propose *Propose) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &propose.shard, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &propose.sig, m); err != nil {
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
	return surge.Unmarshal(r, &propose.latestCommit.Precommits, m)
}

// SizeHint of how many bytes will be needed to represent this prevote in
// binary.
func (prevote Prevote) SizeHint() int {
	return surge.SizeHint(prevote.shard) +
		surge.SizeHint(prevote.signatory) +
		surge.SizeHint(prevote.sig) +
		surge.SizeHint(prevote.height) +
		surge.SizeHint(prevote.round) +
		surge.SizeHint(prevote.blockHash) +
		surge.SizeHint(prevote.nilReasons)
}

// Marshal this prevote into binary.
func (prevote Prevote) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, prevote.shard, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, prevote.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, prevote.sig, m); err != nil {
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

// Unmarshal into this prevote from binary.
func (prevote *Prevote) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &prevote.shard, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &prevote.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &prevote.sig, m); err != nil {
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

// SizeHint of how many bytes will be needed to represent this precommit in
// binary.
func (precommit Precommit) SizeHint() int {
	return surge.SizeHint(precommit.shard) +
		surge.SizeHint(precommit.signatory) +
		surge.SizeHint(precommit.sig) +
		surge.SizeHint(precommit.height) +
		surge.SizeHint(precommit.round) +
		surge.SizeHint(precommit.blockHash)
}

// Marshal this precommit into binary.
func (precommit Precommit) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, precommit.shard, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, precommit.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, precommit.sig, m); err != nil {
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

// Unmarshal into this precommit from binary.
func (precommit *Precommit) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &precommit.shard, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &precommit.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &precommit.sig, m); err != nil {
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

// SizeHint of how many bytes will be needed to represent this resync in
// binary.
func (resync Resync) SizeHint() int {
	return surge.SizeHint(resync.shard) +
		surge.SizeHint(resync.signatory) +
		surge.SizeHint(resync.sig) +
		surge.SizeHint(resync.height) +
		surge.SizeHint(resync.round) +
		surge.SizeHint(resync.timestamp)
}

// Marshal this resync into binary.
func (resync Resync) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, resync.shard, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, resync.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, resync.sig, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, resync.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, resync.round, m); err != nil {
		return m, err
	}
	return surge.Marshal(w, resync.timestamp, m)
}

// Unmarshal into this resync from binary.
func (resync *Resync) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &resync.shard, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &resync.signatory, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &resync.sig, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &resync.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &resync.round, m); err != nil {
		return m, err
	}
	return surge.Unmarshal(r, &resync.timestamp, m)
}

// SizeHint of how many bytes will be needed to represent this inbox in binary.
func (inbox Inbox) SizeHint() int {
	return surge.SizeHint(uint32(inbox.f)) + surge.SizeHint(inbox.messages)
}

// Marshal this inbox into binary.
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
		// Write the number of rounds at this height.
		if m, err = surge.Marshal(w, uint32(len(rounds)), m); err != nil {
			return m, err
		}
		// Write the rounds at this height.
		for round, signatories := range rounds {
			// Write the round.
			if m, err := surge.Marshal(w, round, m); err != nil {
				return m, err
			}
			// Write number of signatories.
			if m, err = surge.Marshal(w, uint32(len(signatories)), m); err != nil {
				return m, err
			}
			// Write signatory/message pairs.
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
	return m, nil
}

// Unmarshal into this inbox from binary.
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
	for i := uint32(0); i < numHeights; i++ {
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
		for j := uint32(0); j < numRounds; j++ {
			// Read the round.
			round := block.Round(0)
			if m, err = surge.Unmarshal(r, &round, m); err != nil {
				return m, fmt.Errorf("cannot unmarshal middle key: %v", err)
			}
			// Read the number of signatories in this round.
			numSignatories := uint32(0)
			if m, err = surge.Unmarshal(r, &numSignatories, m); err != nil {
				return m, fmt.Errorf("cannot unmarshal middle length: %v", err)
			}
			if int(numSignatories) < 0 {
				return m, fmt.Errorf("expected negative length")
			}
			m -= int(numSignatories)
			if m <= 0 {
				return m, surge.ErrMaxBytesExceeded
			}
			inbox.messages[height][round] = map[id.Signatory]Message{}
			for k := uint32(0); k < numSignatories; k++ {
				// Read the signatory.
				signatory := id.Signatory{}
				if m, err = surge.Unmarshal(r, &signatory, m); err != nil {
					return m, fmt.Errorf("cannot unmarshal inner key: %v", err)
				}
				// Read the message from this signatory.
				switch inbox.messageType {
				case ProposeMessageType:
					message := new(Propose)
					if m, err = message.Unmarshal(r, m); err != nil {
						return m, fmt.Errorf("cannot unmarshal propose: %v", err)
					}
					inbox.messages[height][round][signatory] = message
				case PrevoteMessageType:
					message := new(Prevote)
					if m, err = message.Unmarshal(r, m); err != nil {
						return m, fmt.Errorf("cannot unmarshal prevote: %v", err)
					}
					inbox.messages[height][round][signatory] = message
				case PrecommitMessageType:
					message := new(Precommit)
					if m, err = message.Unmarshal(r, m); err != nil {
						return m, fmt.Errorf("cannot unmarshal precommit: %v", err)
					}
					inbox.messages[height][round][signatory] = message
				default:
					return m, fmt.Errorf("unsupported MessageType=%v", inbox.messageType)
				}
			}
		}
	}
	return m, nil
}

// SizeHint of how many bytes will be needed to represent state in binary.
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

// Marshal this state into binary.
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

// Unmarshal into this state from binary.
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
	state.Proposals = new(Inbox)
	if m, err = surge.Unmarshal(r, state.Proposals, m); err != nil {
		return m, err
	}
	state.Prevotes = new(Inbox)
	if m, err = surge.Unmarshal(r, state.Prevotes, m); err != nil {
		return m, err
	}
	state.Precommits = new(Inbox)
	return surge.Unmarshal(r, state.Precommits, m)
}

// MarshalMessage into binary.
func MarshalMessage(msg Message, w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, uint64(msg.Type()), m)
	if err != nil {
		return m, err
	}
	return surge.Marshal(w, msg, m)
}

// UnmarshalMessage into this message from binary.
func UnmarshalMessage(r io.Reader, m int) (Message, int, error) {
	var messageType MessageType
	m, err := surge.Unmarshal(r, &messageType, m)
	if err != nil {
		return nil, m, err
	}

	var msg Message
	switch messageType {
	case ProposeMessageType:
		propose := new(Propose)
		m, err = propose.Unmarshal(r, m)
		msg = propose
	case PrevoteMessageType:
		prevote := new(Prevote)
		m, err = prevote.Unmarshal(r, m)
		msg = prevote
	case PrecommitMessageType:
		precommit := new(Precommit)
		m, err = precommit.Unmarshal(r, m)
		msg = precommit
	case ResyncMessageType:
		resync := new(Resync)
		m, err = resync.Unmarshal(r, m)
		msg = resync
	default:
		return nil, m, fmt.Errorf("unexpected message type %d", messageType)
	}
	return msg, m, err
}
