package process

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
)

// MarshalJSON implements the `json.Marshaler` interface for the Propose type.
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

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Propose
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

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// Propose type.
func (propose Propose) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, propose.sig); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.sig: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, propose.signatory); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.signatory: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, propose.height); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.height: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, propose.round); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.round: %v", err)
	}
	blockData, err := propose.block.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal propose.block: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(blockData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.block len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, blockData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.block data: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, propose.validRound); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.validRound: %v", err)
	}
	latestCommitBlockData, err := propose.latestCommit.Block.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal propose.latestCommit.Block: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(latestCommitBlockData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.latestCommit.Block len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, latestCommitBlockData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.latestCommit.Block data: %v", err)
	}
	lenPrecommits := len(propose.latestCommit.Precommits)
	if err := binary.Write(buf, binary.LittleEndian, uint64(lenPrecommits)); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write propose.latestCommit.Precommits len: %v", err)
	}
	for i := 0; i < lenPrecommits; i++ {
		latestPrecommitBytes, err := propose.latestCommit.Precommits[i].MarshalBinary()
		if err != nil {
			return buf.Bytes(), fmt.Errorf("cannot marshal propose.latestCommit precommit: %v", err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint64(len(latestPrecommitBytes))); err != nil {
			return buf.Bytes(), fmt.Errorf("cannot write propose.latestCommit precommit len: %v", err)
		}
		if err := binary.Write(buf, binary.LittleEndian, latestPrecommitBytes); err != nil {
			return buf.Bytes(), fmt.Errorf("cannot write propose.latestCommit precommit data: %v", err)
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// Propose type.
func (propose *Propose) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.LittleEndian, &propose.sig); err != nil {
		return fmt.Errorf("cannot read propose.sig: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &propose.signatory); err != nil {
		return fmt.Errorf("cannot read propose.signatory: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &propose.height); err != nil {
		return fmt.Errorf("cannot read propose.height: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &propose.round); err != nil {
		return fmt.Errorf("cannot read propose.round: %v", err)
	}
	var numBytes uint64
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read propose.block len: %v", err)
	}
	blockBytes := make([]byte, numBytes)
	if _, err := buf.Read(blockBytes); err != nil {
		return fmt.Errorf("cannot read propose.block data: %v", err)
	}
	if err := propose.block.UnmarshalBinary(blockBytes); err != nil {
		return fmt.Errorf("cannot unmarshal propose.block: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &propose.validRound); err != nil {
		return fmt.Errorf("cannot read propose.validRound: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read propose.latestCommit.Block len: %v", err)
	}
	latestCommitBlockBytes := make([]byte, numBytes)
	if _, err := buf.Read(latestCommitBlockBytes); err != nil {
		return fmt.Errorf("cannot read propose.latestCommit.Block data: %v", err)
	}
	if err := propose.latestCommit.Block.UnmarshalBinary(latestCommitBlockBytes); err != nil {
		return fmt.Errorf("cannot unmarshal propose.latestCommit.Block: %v", err)
	}
	var lenPrecommits uint64
	if err := binary.Read(buf, binary.LittleEndian, &lenPrecommits); err != nil {
		return fmt.Errorf("cannot read propose.latestCommit.Precommits len: %v", err)
	}
	if lenPrecommits > 0 {
		propose.latestCommit.Precommits = make([]Precommit, lenPrecommits)
	}
	for i := uint64(0); i < lenPrecommits; i++ {
		if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
			return fmt.Errorf("cannot read propose.latestCommit precommit len: %v", err)
		}
		latestPrecommitBlockBytes := make([]byte, numBytes)
		if _, err := buf.Read(latestPrecommitBlockBytes); err != nil {
			return fmt.Errorf("cannot read propose.latestCommit precommit data: %v", err)
		}
		if err := propose.latestCommit.Precommits[i].UnmarshalBinary(latestPrecommitBlockBytes); err != nil {
			return fmt.Errorf("cannot unmarshal propose.latestCommit precommit: %v", err)
		}
	}
	return nil
}

// MarshalJSON implements the `json.Marshaler` interface for the Prevote type.
func (prevote Prevote) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sig       id.Signature `json:"sig"`
		Signatory id.Signatory `json:"signatory"`
		Height    block.Height `json:"height"`
		Round     block.Round  `json:"round"`
		BlockHash id.Hash      `json:"blockHash"`
	}{
		prevote.sig,
		prevote.signatory,
		prevote.height,
		prevote.round,
		prevote.blockHash,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Prevote type.
func (prevote *Prevote) UnmarshalJSON(data []byte) error {
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
	prevote.sig = tmp.Sig
	prevote.signatory = tmp.Signatory
	prevote.height = tmp.Height
	prevote.round = tmp.Round
	prevote.blockHash = tmp.BlockHash
	return nil
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// Prevote type.
func (prevote Prevote) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, prevote.sig); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write prevote.sig: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, prevote.signatory); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write prevote.signatory: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, prevote.height); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write prevote.height: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, prevote.round); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write prevote.round: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, prevote.blockHash); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write prevote.blockHash: %v", err)
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// Prevote type.
func (prevote *Prevote) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.LittleEndian, &prevote.sig); err != nil {
		return fmt.Errorf("cannot read prevote.sig: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &prevote.signatory); err != nil {
		return fmt.Errorf("cannot read prevote.signatory: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &prevote.height); err != nil {
		return fmt.Errorf("cannot read prevote.height: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &prevote.round); err != nil {
		return fmt.Errorf("cannot read prevote.round: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &prevote.blockHash); err != nil {
		return fmt.Errorf("cannot read prevote.blockHash: %v", err)
	}
	return nil
}

// MarshalJSON implements the `json.Marshaler` interface for the Precommit type.
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

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Precommit type.
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

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// Precommit type.
func (precommit Precommit) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, precommit.sig); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write precommit.sig: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, precommit.signatory); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write precommit.signatory: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, precommit.height); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write precommit.height: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, precommit.round); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write precommit.round: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, precommit.blockHash); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write precommit.blockHash: %v", err)
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// Precommit type.
func (precommit *Precommit) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.LittleEndian, &precommit.sig); err != nil {
		return fmt.Errorf("cannot read precommit.sig: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &precommit.signatory); err != nil {
		return fmt.Errorf("cannot read precommit.signatory: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &precommit.height); err != nil {
		return fmt.Errorf("cannot read precommit.height: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &precommit.round); err != nil {
		return fmt.Errorf("cannot read precommit.round: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &precommit.blockHash); err != nil {
		return fmt.Errorf("cannot read precommit.blockHash: %v", err)
	}

	return nil
}

// MarshalJSON implements the `json.Marshaler` interface for the Inbox type.
func (inbox Inbox) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		F        int                                                       `json:"f"`
		Messages map[block.Height]map[block.Round]map[id.Signatory]Message `json:"messages"`
	}{
		inbox.f,
		inbox.messages,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Inbox type.
// Note : you need to be really careful when doing unmarshaling, specifically you need
// to initialize the inbox with the expected messageType. Otherwise it would panic.
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

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// Inbox type.
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
// Inbox type.
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
