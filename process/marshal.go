package process

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
	"github.com/renproject/surge"
)

// MarshalJSON implements the `json.Marshaler` interface for the `Propose` type.
func (propose Propose) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sig          id.Signature `json:"sig"`
		Signatory    id.Signatory `json:"signatory"`
		Height       block.Height `json:"height"`
		Round        block.Round  `json:"round"`
		Block        block.Block  `json:"block"`
		ValidRound   block.Round  `json:"validRound"`
		LatestCommit LatestCommit `json:"latestCommit"`
	}{
		propose.sig,
		propose.signatory,
		propose.height,
		propose.round,
		propose.block,
		propose.validRound,
		propose.latestCommit,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the `Propose`
// type.
func (propose *Propose) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Sig          id.Signature `json:"sig"`
		Signatory    id.Signatory `json:"signatory"`
		Height       block.Height `json:"height"`
		Round        block.Round  `json:"round"`
		Block        block.Block  `json:"block"`
		ValidRound   block.Round  `json:"validRound"`
		LatestCommit LatestCommit `json:"latestCommit"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	propose.sig = tmp.Sig
	propose.signatory = tmp.Signatory
	propose.height = tmp.Height
	propose.round = tmp.Round
	propose.block = tmp.Block
	propose.validRound = tmp.ValidRound
	propose.latestCommit = tmp.LatestCommit
	return nil
}

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

// MarshalJSON implements the `json.Marshaler` interface for the `Prevote` type.
func (prevote Prevote) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sig        id.Signature `json:"sig"`
		Signatory  id.Signatory `json:"signatory"`
		Height     block.Height `json:"height"`
		Round      block.Round  `json:"round"`
		BlockHash  id.Hash      `json:"blockHash"`
		NilReasons NilReasons   `json:"nilReasons"`
	}{
		prevote.sig,
		prevote.signatory,
		prevote.height,
		prevote.round,
		prevote.blockHash,
		prevote.nilReasons,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the `Prevote`
// type.
func (prevote *Prevote) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Sig        id.Signature `json:"sig"`
		Signatory  id.Signatory `json:"signatory"`
		Height     block.Height `json:"height"`
		Round      block.Round  `json:"round"`
		BlockHash  id.Hash      `json:"blockHash"`
		NilReasons NilReasons   `json:"nilReasons"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	prevote.sig = tmp.Sig
	prevote.signatory = tmp.Signatory
	prevote.height = tmp.Height
	prevote.round = tmp.Round
	prevote.blockHash = tmp.BlockHash
	prevote.nilReasons = tmp.NilReasons
	return nil
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

// MarshalJSON implements the `json.Marshaler` interface for the `Precommit`
// type.
func (precommit Precommit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sig       id.Signature `json:"sig"`
		Signatory id.Signatory `json:"signatory"`
		Height    block.Height `json:"height"`
		Round     block.Round  `json:"round"`
		BlockHash id.Hash      `json:"blockHash"`
	}{
		precommit.sig,
		precommit.signatory,
		precommit.height,
		precommit.round,
		precommit.blockHash,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the `Precommit`
// type.
func (precommit *Precommit) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Sig       id.Signature `json:"sig"`
		Signatory id.Signatory `json:"signatory"`
		Height    block.Height `json:"height"`
		Round     block.Round  `json:"round"`
		BlockHash id.Hash      `json:"blockHash"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	precommit.sig = tmp.Sig
	precommit.signatory = tmp.Signatory
	precommit.height = tmp.Height
	precommit.round = tmp.Round
	precommit.blockHash = tmp.BlockHash
	return nil
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

// MarshalJSON implements the `json.Marshaler` interface for the `Inbox` type.
func (inbox Inbox) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		F        int                                                       `json:"f"`
		Messages map[block.Height]map[block.Round]map[id.Signatory]Message `json:"messages"`
	}{
		inbox.f,
		inbox.messages,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the `Inbox`
// type. Before unmarshaling into an inbox, you must initialise it. Unmarshaling
// will panic if the inbox in not initialised, or if it is initialised with the
// wrong message type.
func (inbox *Inbox) UnmarshalJSON(data []byte) error {
	tmp := struct {
		F        int                                                               `json:"f"`
		Messages map[block.Height]map[block.Round]map[id.Signatory]json.RawMessage `json:"messages"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	inbox.f = tmp.F
	inbox.messages = map[block.Height]map[block.Round]map[id.Signatory]Message{}

	for height, roundMap := range tmp.Messages {
		if roundMap != nil {
			inbox.messages[height] = map[block.Round]map[id.Signatory]Message{}
		}
		for round, sigMap := range roundMap {
			if sigMap != nil {
				inbox.messages[height][round] = map[id.Signatory]Message{}
			}
			for sig, raw := range sigMap {
				var err error
				switch inbox.messageType {
				case ProposeMessageType:
					msg := new(Propose)
					err = json.Unmarshal(raw, msg)
					inbox.messages[height][round][sig] = msg
				case PrevoteMessageType:
					msg := new(Prevote)
					err = json.Unmarshal(raw, msg)
					inbox.messages[height][round][sig] = msg
				case PrecommitMessageType:
					msg := new(Precommit)
					err = json.Unmarshal(raw, msg)
					inbox.messages[height][round][sig] = msg
				}
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (inbox Inbox) SizeHint() int {
	return surge.SizeHint(uint32(inbox.f)) + surge.SizeHint(inbox.messages)
}

func (inbox Inbox) Marshal(w io.Writer, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	m, err := surge.Marshal(w, uint32(inbox.f), m)
	if err != nil {
		return m, err
	}
	return surge.Marshal(w, inbox.messages, m)
}

func (inbox *Inbox) Unmarshal(r io.Reader, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	var f uint32
	m, err := surge.Unmarshal(r, &f, m)
	if err != nil {
		return m, err
	}
	inbox.f = int(f)
	return surge.Unmarshal(r, &inbox.messages, m)
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// `Inbox` type.
func (inbox Inbox) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint64(inbox.f)); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write inbox.f: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(inbox.messages))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write inbox.messages len: %v", err)
	}
	for height, roundMap := range inbox.messages {
		if err := binary.Write(buf, binary.LittleEndian, height); err != nil {
			return buf.Bytes(), fmt.Errorf("cannot write inbox.messages height: %v", err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint64(len(roundMap))); err != nil {
			return buf.Bytes(), fmt.Errorf("cannot write inbox.messages roundMap len: %v", err)
		}
		for round, sigMap := range roundMap {
			if err := binary.Write(buf, binary.LittleEndian, round); err != nil {
				return buf.Bytes(), fmt.Errorf("cannot write inbox.messages round: %v", err)
			}
			if err := binary.Write(buf, binary.LittleEndian, uint64(len(sigMap))); err != nil {
				return buf.Bytes(), fmt.Errorf("cannot write inbox.messages sigMap len: %v", err)
			}
			for sig, message := range sigMap {
				if err := binary.Write(buf, binary.LittleEndian, sig); err != nil {
					return buf.Bytes(), fmt.Errorf("cannot write inbox.messages sig: %v", err)
				}
				messageData, err := message.MarshalBinary()
				if err != nil {
					return buf.Bytes(), fmt.Errorf("cannot marshal message: %v", err)
				}
				if err := binary.Write(buf, binary.LittleEndian, uint64(len(messageData))); err != nil {
					return buf.Bytes(), fmt.Errorf("cannot write message len: %v", err)
				}
				if err := binary.Write(buf, binary.LittleEndian, messageData); err != nil {
					return buf.Bytes(), fmt.Errorf("cannot write message data: %v", err)
				}
			}
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// `Inbox` type. See the `UnmarshalJSON` method for more information.
func (inbox *Inbox) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	var f uint64
	if err := binary.Read(buf, binary.LittleEndian, &f); err != nil {
		return fmt.Errorf("cannot read inbox.f: %v", err)
	}
	inbox.f = int(f)
	var heightMapLen uint64
	if err := binary.Read(buf, binary.LittleEndian, &heightMapLen); err != nil {
		return fmt.Errorf("cannot read inbox.messages len: %v", err)
	}
	heightMap := make(map[block.Height]map[block.Round]map[id.Signatory]Message, heightMapLen)
	for i := uint64(0); i < heightMapLen; i++ {
		var height block.Height
		if err := binary.Read(buf, binary.LittleEndian, &height); err != nil {
			return fmt.Errorf("cannot read inbox.messages height: %v", err)
		}
		var roundMapLen uint64
		if err := binary.Read(buf, binary.LittleEndian, &roundMapLen); err != nil {
			return fmt.Errorf("cannot read inbox.messages roundMap len: %v", err)
		}
		roundMap := make(map[block.Round]map[id.Signatory]Message, roundMapLen)
		for j := uint64(0); j < roundMapLen; j++ {
			var round block.Round
			if err := binary.Read(buf, binary.LittleEndian, &round); err != nil {
				return fmt.Errorf("cannot read inbox.messages round: %v", err)
			}
			var sigMapLen uint64
			if err := binary.Read(buf, binary.LittleEndian, &sigMapLen); err != nil {
				return fmt.Errorf("cannot read inbox.messages sigMap len: %v", err)
			}
			sigMap := make(map[id.Signatory]Message, sigMapLen)
			for k := uint64(0); k < sigMapLen; k++ {
				var sig id.Signatory
				if err := binary.Read(buf, binary.LittleEndian, &sig); err != nil {
					return fmt.Errorf("cannot read inbox.messages sig: %v", err)
				}
				var numBytes uint64
				if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
					return fmt.Errorf("cannot read inbox.messages message len: %v", err)
				}
				messageBytes := make([]byte, numBytes)
				if _, err := buf.Read(messageBytes); err != nil {
					return fmt.Errorf("cannot read inbox.messages message data: %v", err)
				}

				var err error
				switch inbox.messageType {
				case ProposeMessageType:
					message := new(Propose)
					err = message.UnmarshalBinary(messageBytes)
					sigMap[sig] = message
				case PrevoteMessageType:
					message := new(Prevote)
					err = message.UnmarshalBinary(messageBytes)
					sigMap[sig] = message
				case PrecommitMessageType:
					message := new(Precommit)
					err = message.UnmarshalBinary(messageBytes)
					sigMap[sig] = message
				}
				if err != nil {
					return fmt.Errorf("cannot unmarshal inbox.messages message: %v", err)
				}
			}
			roundMap[round] = sigMap
		}
		heightMap[height] = roundMap
	}
	inbox.messages = heightMap
	return nil
}

func (state State) Marshal(w io.Writer, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

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
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

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
	if m, err = surge.Unmarshal(r, &state.Proposals, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &state.Prevotes, m); err != nil {
		return m, err
	}
	return surge.Unmarshal(r, &state.Precommits, m)
}
