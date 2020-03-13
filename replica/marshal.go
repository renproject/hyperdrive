package replica

import (
	"fmt"
	"io"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/surge"
)

func (message Message) SizeHint() int {
	return surge.SizeHint(message.Message.Type()) +
		surge.SizeHint(message.Message) +
		surge.SizeHint(message.Shard)
}

func (message Message) Marshal(w io.Writer, m int) (int, error) {
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
	var messageType process.MessageType
	m, err := surge.Unmarshal(r, &messageType, m)
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
	case process.ResyncMessageType:
		resync := new(process.Resync)
		m, err = resync.Unmarshal(r, m)
		message.Message = resync
	default:
		return m, fmt.Errorf("unexpected message type %d", messageType)
	}
	if err != nil {
		return m, err
	}

	return surge.Unmarshal(r, &message.Shard, m)
}

func (message Message) MarshalBinary() ([]byte, error) {
	return surge.ToBinary(message)
}

func (message *Message) UnmarshalBinary(data []byte) error {
	return surge.FromBinary(data, message)
}
