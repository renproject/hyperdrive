package replica

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/renproject/hyperdrive/process"
)

func (m Message) MarshalJSON() ([]byte, error) {
	tmp := struct {
		MessageType process.MessageType `json:"type"`
		Message     process.Message     `json:"message"`
		Shard       Shard               `json:"shard"`
	}{
		MessageType: m.Message.Type(),
		Message:     m.Message,
		Shard:       m.Shard,
	}
	return json.Marshal(tmp)
}

func (m *Message) UnmarshalJSON(data []byte) error {
	tmp := struct {
		MessageType process.MessageType `json:"type"`
		Message     json.RawMessage     `json:"message"`
		Shard       Shard               `json:"shard"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	switch tmp.MessageType {
	case process.ProposeMessageType:
		propose := new(process.Propose)
		if err := propose.UnmarshalJSON(tmp.Message); err != nil {
			return err
		}
		m.Message = propose
	case process.PrevoteMessageType:
		prevote := new(process.Prevote)
		if err := prevote.UnmarshalJSON(tmp.Message); err != nil {
			return err
		}
		m.Message = prevote
	case process.PrecommitMessageType:
		precommit := new(process.Precommit)
		if err := precommit.UnmarshalJSON(tmp.Message); err != nil {
			return err
		}
		m.Message = precommit
	}
	m.Shard = tmp.Shard

	return nil
}

func (m Message) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	messageData, err := m.Message.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal m.Message: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(messageData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write m.Message len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, m.Message.Type()); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write m.Message.Type: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, messageData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write m.Message data: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, m.Shard); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write m.Shard: %v", err)
	}
	return buf.Bytes(), nil
}

func (m *Message) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	var numBytes uint64
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read m.Message len: %v", err)
	}
	var messageType uint64
	if err := binary.Read(buf, binary.LittleEndian, &messageType); err != nil {
		return fmt.Errorf("cannot read m.Message.Type: %v", err)
	}
	messageBytes := make([]byte, numBytes)
	if _, err := buf.Read(messageBytes); err != nil {
		return fmt.Errorf("cannot read m.Message data: %v", err)
	}
	var err error
	switch messageType {
	case process.ProposeMessageType:
		propose := new(process.Propose)
		err = propose.UnmarshalBinary(messageBytes)
		m.Message = propose
	case process.PrevoteMessageType:
		prevote := new(process.Prevote)
		err = prevote.UnmarshalBinary(messageBytes)
		m.Message = prevote
	case process.PrecommitMessageType:
		precommit := new(process.Precommit)
		err = precommit.UnmarshalBinary(messageBytes)
		m.Message = precommit
	default:
		return fmt.Errorf("unexpected message type %d", messageType)
	}
	if err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &m.Shard); err != nil {
		return fmt.Errorf("cannot read m.Shard: %v", err)
	}
	return nil
}
