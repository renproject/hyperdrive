package replica

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/surge"
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

func (message Message) SizeHint() int {
	return surge.SizeHint(message.Message.Type()) +
		surge.SizeHint(message.Message) +
		surge.SizeHint(message.Shard)
}

func (message Message) Marshal(w io.Writer, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	m, err := surge.Marshal(w, uint64(message.Message.Type()), m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, message.Message, m); err != nil {
		return m, err
	}
	return surge.Marshal(w, message.Shard, m)
}

func (message *Message) Unmarshal(r io.Reader, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

	var messageType process.MessageType
	m, err := surge.Unmarshal(r, (*uint64)(&messageType), m)
	if err != nil {
		return m, err
	}

	switch messageType {
	case process.ProposeMessageType:
		propose := new(process.Propose)
		m, err = propose.Unmarshal(r, m)
		message.Message = propose
	case process.PrevoteMessageType:
		prevote := new(process.Prevote)
		m, err = prevote.Unmarshal(r, m)
		message.Message = prevote
	case process.PrecommitMessageType:
		precommit := new(process.Precommit)
		m, err = precommit.Unmarshal(r, m)
		message.Message = precommit
	default:
		return m, fmt.Errorf("unexpected message type %d", messageType)
	}
	if err != nil {
		return m, err
	}

	return surge.Unmarshal(r, message.Shard, m)
}

func (message Message) MarshalBinary() ([]byte, error) {
	return surge.ToBinary(message)
}

func (message *Message) UnmarshalBinary(data []byte) error {
	return surge.FromBinary(data, message)
}
