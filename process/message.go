package process

import (
	"fmt"

	"github.com/renproject/id"
	"github.com/renproject/surge"
)

// A Propose message is sent by the proposer Process at most once per Round. The
// Scheduler interfaces determines which Process is the proposer at any given
// Height and Round.
type Propose struct {
	Height     Height       `json:"height"`
	Round      Round        `json:"round"`
	ValidRound Round        `json:"validRound"`
	Value      Value        `json:"value"`
	From       id.Signatory `json:"from"`
	Signature  id.Signature `json:"signature"`
}

// NewProposeHash receives fields of a propose message and hashes the message
func NewProposeHash(height Height, round Round, validRound Round, value Value) (id.Hash, error) {
	sizeHint := surge.SizeHint(height) + surge.SizeHint(round) + surge.SizeHint(validRound) + surge.SizeHint(value)
	buf := make([]byte, sizeHint)
	return NewProposeHashWithBuffer(height, round, validRound, value, buf)
}

// NewProposeHashWithBuffer receives fields of a propose message, with a bytes buffer and hashes the message
func NewProposeHashWithBuffer(height Height, round Round, validRound Round, value Value, data []byte) (id.Hash, error) {
	buf, rem, err := surge.Marshal(height, data, surge.MaxBytes)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling height=%v: %v", height, err)
	}
	buf, rem, err = surge.Marshal(round, buf, rem)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling round=%v: %v", round, err)
	}
	buf, rem, err = surge.Marshal(validRound, buf, rem)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling valid round=%v: %v", validRound, err)
	}
	buf, rem, err = surge.Marshal(value, buf, rem)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling value=%v: %v", value, err)
	}
	return id.NewHash(data), nil
}

// Equal compares two Proposes. If they are equal, then it return true,
// otherwise it returns false. The signatures are not checked for equality,
// because signatures include randomness.
func (propose Propose) Equal(other *Propose) bool {
	return propose.Height == other.Height &&
		propose.Round == other.Round &&
		propose.ValidRound == other.ValidRound &&
		propose.Value.Equal(&other.Value) &&
		propose.From.Equal(&other.From)
}

// SizeHint returns the number of bytes required to represent this message in
// binary.
func (propose Propose) SizeHint() int {
	return surge.SizeHint(propose.Height) +
		surge.SizeHint(propose.Round) +
		surge.SizeHint(propose.ValidRound) +
		surge.SizeHint(propose.Value) +
		surge.SizeHint(propose.From) +
		surge.SizeHint(propose.Signature)
}

// Marshal this message into binary.
func (propose Propose) Marshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Marshal(propose.Height, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling height=%v: %v", propose.Height, err)
	}
	buf, rem, err = surge.Marshal(propose.Round, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling round=%v: %v", propose.Round, err)
	}
	buf, rem, err = surge.Marshal(propose.ValidRound, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling valid round=%v: %v", propose.ValidRound, err)
	}
	buf, rem, err = surge.Marshal(propose.Value, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling value=%v: %v", propose.Value, err)
	}
	buf, rem, err = surge.Marshal(propose.From, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling from=%v: %v", propose.From, err)
	}
	buf, rem, err = surge.Marshal(propose.Signature, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling signature=%v: %v", propose.Signature, err)
	}
	return buf, rem, nil
}

// Unmarshal binary into this message.
func (propose *Propose) Unmarshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Unmarshal(&propose.Height, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling height: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&propose.Round, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling round: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&propose.ValidRound, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling valid round: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&propose.Value, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling value: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&propose.From, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling from: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&propose.Signature, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling signature: %v", err)
	}
	return buf, rem, nil
}

// A Prevote is sent by every correct Process at most once per Round. It is the
// first step of reaching consensus. Informally, if a correct Process receives
// 2F+1 Precommits for a Value, then it will Precommit to that Value. However,
// there are many other conditions which can cause a Process to Prevote. See the
// Process for more information.
type Prevote struct {
	Height    Height       `json:"height"`
	Round     Round        `json:"round"`
	Value     Value        `json:"value"`
	From      id.Signatory `json:"from"`
	Signature id.Signature `json:"signature"`
}

// NewPrevoteHash receives fields of a prevote message and hashes the message
func NewPrevoteHash(height Height, round Round, value Value) (id.Hash, error) {
	sizeHint := surge.SizeHint(height) + surge.SizeHint(round) + surge.SizeHint(value)
	buf := make([]byte, sizeHint)
	return NewPrevoteHashWithBuffer(height, round, value, buf)
}

// NewPrevoteHashWithBuffer receives fields of a prevote message, with a bytes buffer and hashes the message
func NewPrevoteHashWithBuffer(height Height, round Round, value Value, data []byte) (id.Hash, error) {
	buf, rem, err := surge.Marshal(height, data, surge.MaxBytes)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling height=%v: %v", height, err)
	}
	buf, rem, err = surge.Marshal(round, buf, rem)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling round=%v: %v", round, err)
	}
	buf, rem, err = surge.Marshal(value, buf, rem)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling value=%v: %v", value, err)
	}
	return id.NewHash(data), nil
}

// Equal compares two Prevotes. If they are equal, then it return true,
// otherwise it returns false. The signatures are not checked for equality,
// because signatures include randomness.
func (prevote Prevote) Equal(other *Prevote) bool {
	return prevote.Height == other.Height &&
		prevote.Round == other.Round &&
		prevote.Value.Equal(&other.Value) &&
		prevote.From.Equal(&other.From)
}

// SizeHint returns the number of bytes required to represent this message in
// binary.
func (prevote Prevote) SizeHint() int {
	return surge.SizeHint(prevote.Height) +
		surge.SizeHint(prevote.Round) +
		surge.SizeHint(prevote.Value) +
		surge.SizeHint(prevote.From) +
		surge.SizeHint(prevote.Signature)
}

// Marshal this message into binary.
func (prevote Prevote) Marshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Marshal(prevote.Height, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling height=%v: %v", prevote.Height, err)
	}
	buf, rem, err = surge.Marshal(prevote.Round, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling round=%v: %v", prevote.Round, err)
	}
	buf, rem, err = surge.Marshal(prevote.Value, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling value=%v: %v", prevote.Value, err)
	}
	buf, rem, err = surge.Marshal(prevote.From, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling from=%v: %v", prevote.From, err)
	}
	buf, rem, err = surge.Marshal(prevote.Signature, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling signature=%v: %v", prevote.Signature, err)
	}
	return buf, rem, nil
}

// Unmarshal binary into this message.
func (prevote *Prevote) Unmarshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Unmarshal(&prevote.Height, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling height: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&prevote.Round, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling round: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&prevote.Value, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling value: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&prevote.From, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling from: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&prevote.Signature, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling signature: %v", err)
	}
	return buf, rem, nil
}

// A Precommit is sent by every correct Process at most once per Round. It is
// the second step of reaching consensus. Informally, if a correct Process
// receives 2F+1 Precommits for a Value, then it will commit to that Value and
// progress to the next Height. However, there are many other conditions which
// can cause a Process to Precommit. See the Process for more information.
type Precommit struct {
	Height    Height       `json:"height"`
	Round     Round        `json:"round"`
	Value     Value        `json:"value"`
	From      id.Signatory `json:"from"`
	Signature id.Signature `json:"signature"`
}

// NewPrecommitHash receives fields of a precommit message and hashes the message
func NewPrecommitHash(height Height, round Round, value Value) (id.Hash, error) {
	sizeHint := surge.SizeHint(height) + surge.SizeHint(round) + surge.SizeHint(value)
	buf := make([]byte, sizeHint)
	return NewPrecommitHashWithBuffer(height, round, value, buf)
}

// NewPrecommitHashWithBuffer receives fields of a precommit message, with a bytes buffer and hashes the message
func NewPrecommitHashWithBuffer(height Height, round Round, value Value, data []byte) (id.Hash, error) {
	buf, rem, err := surge.Marshal(height, data, surge.MaxBytes)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling height=%v: %v", height, err)
	}
	buf, rem, err = surge.Marshal(round, buf, rem)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling round=%v: %v", round, err)
	}
	buf, rem, err = surge.Marshal(value, buf, rem)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling value=%v: %v", value, err)
	}
	return id.NewHash(data), nil
}

// Equal compares two Precommits. If they are equal, then it return true,
// otherwise it returns false. The signatures are not checked for equality,
// because signatures include randomness.
func (precommit Precommit) Equal(other *Precommit) bool {
	return precommit.Height == other.Height &&
		precommit.Round == other.Round &&
		precommit.Value.Equal(&other.Value) &&
		precommit.From.Equal(&other.From)
}

// SizeHint returns the number of bytes required to represent this message in
// binary.
func (precommit Precommit) SizeHint() int {
	return surge.SizeHint(precommit.Height) +
		surge.SizeHint(precommit.Round) +
		surge.SizeHint(precommit.Value) +
		surge.SizeHint(precommit.From) +
		surge.SizeHint(precommit.Signature)
}

// Marshal this message into binary.
func (precommit Precommit) Marshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Marshal(precommit.Height, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling height=%v: %v", precommit.Height, err)
	}
	buf, rem, err = surge.Marshal(precommit.Round, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling round=%v: %v", precommit.Round, err)
	}
	buf, rem, err = surge.Marshal(precommit.Value, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling value=%v: %v", precommit.Value, err)
	}
	buf, rem, err = surge.Marshal(precommit.From, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling from=%v: %v", precommit.From, err)
	}
	buf, rem, err = surge.Marshal(precommit.Signature, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling signature=%v: %v", precommit.Signature, err)
	}
	return buf, rem, nil
}

// Unmarshal binary into this message.
func (precommit *Precommit) Unmarshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Unmarshal(&precommit.Height, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling height: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&precommit.Round, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling round: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&precommit.Value, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling value: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&precommit.From, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling from: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&precommit.Signature, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling signature: %v", err)
	}
	return buf, rem, nil
}
