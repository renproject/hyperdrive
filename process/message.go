package process

import (
	"bytes"
	"fmt"
	"io"

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

func NewProposeHash(height Height, round Round, validRound Round, value Value) (id.Hash, error) {
	buf := new(bytes.Buffer)
	buf.Grow(surge.SizeHint(height) + surge.SizeHint(round) + surge.SizeHint(validRound) + surge.SizeHint(value))
	return NewProposeHashWithBuffer(height, round, validRound, value, buf)
}

func NewProposeHashWithBuffer(height Height, round Round, validRound Round, value Value, buf *bytes.Buffer) (id.Hash, error) {
	m, err := surge.Marshal(buf, height, surge.MaxBytes)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling height=%v: %v", height, err)
	}
	m, err = surge.Marshal(buf, round, m)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling round=%v: %v", round, err)
	}
	m, err = surge.Marshal(buf, validRound, m)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling valid round=%v: %v", validRound, err)
	}
	m, err = surge.Marshal(buf, value, m)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling value=%v: %v", value, err)
	}
	return id.NewHash(buf.Bytes()), nil
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
func (propose Propose) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, propose.Height, m)
	if err != nil {
		return m, fmt.Errorf("marshaling height=%v: %v", propose.Height, err)
	}
	m, err = surge.Marshal(w, propose.Round, m)
	if err != nil {
		return m, fmt.Errorf("marshaling round=%v: %v", propose.Round, err)
	}
	m, err = surge.Marshal(w, propose.ValidRound, m)
	if err != nil {
		return m, fmt.Errorf("marshaling valid round=%v: %v", propose.ValidRound, err)
	}
	m, err = surge.Marshal(w, propose.Value, m)
	if err != nil {
		return m, fmt.Errorf("marshaling value=%v: %v", propose.Value, err)
	}
	m, err = surge.Marshal(w, propose.From, m)
	if err != nil {
		return m, fmt.Errorf("marshaling from=%v: %v", propose.From, err)
	}
	m, err = surge.Marshal(w, propose.Signature, m)
	if err != nil {
		return m, fmt.Errorf("marshaling signature=%v: %v", propose.Signature, err)
	}
	return m, nil
}

// Unmarshal binary into this message.
func (propose *Propose) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &propose.Height, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling height: %v", err)
	}
	m, err = surge.Unmarshal(r, &propose.Round, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling round: %v", err)
	}
	m, err = surge.Unmarshal(r, &propose.ValidRound, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling valid round: %v", err)
	}
	m, err = surge.Unmarshal(r, &propose.Value, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling value: %v", err)
	}
	m, err = surge.Unmarshal(r, &propose.From, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling from: %v", err)
	}
	m, err = surge.Unmarshal(r, &propose.Signature, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling signature: %v", err)
	}
	return m, nil
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

func NewPrevoteHash(height Height, round Round, value Value) (id.Hash, error) {
	buf := new(bytes.Buffer)
	buf.Grow(surge.SizeHint(height) + surge.SizeHint(round) + surge.SizeHint(value))
	return NewPrevoteHashWithBuffer(height, round, value, buf)
}

func NewPrevoteHashWithBuffer(height Height, round Round, value Value, buf *bytes.Buffer) (id.Hash, error) {
	m, err := surge.Marshal(buf, height, surge.MaxBytes)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling height=%v: %v", height, err)
	}
	m, err = surge.Marshal(buf, round, m)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling round=%v: %v", round, err)
	}
	m, err = surge.Marshal(buf, value, m)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling value=%v: %v", value, err)
	}
	return id.NewHash(buf.Bytes()), nil
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
func (prevote Prevote) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, prevote.Height, m)
	if err != nil {
		return m, fmt.Errorf("marshaling height=%v: %v", prevote.Height, err)
	}
	m, err = surge.Marshal(w, prevote.Round, m)
	if err != nil {
		return m, fmt.Errorf("marshaling round=%v: %v", prevote.Round, err)
	}
	m, err = surge.Marshal(w, prevote.Value, m)
	if err != nil {
		return m, fmt.Errorf("marshaling value=%v: %v", prevote.Value, err)
	}
	m, err = surge.Marshal(w, prevote.From, m)
	if err != nil {
		return m, fmt.Errorf("marshaling from=%v: %v", prevote.From, err)
	}
	m, err = surge.Marshal(w, prevote.Signature, m)
	if err != nil {
		return m, fmt.Errorf("marshaling signature=%v: %v", prevote.Signature, err)
	}
	return m, nil
}

// Unmarshal binary into this message.
func (prevote *Prevote) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &prevote.Height, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling height: %v", err)
	}
	m, err = surge.Unmarshal(r, &prevote.Round, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling round: %v", err)
	}
	m, err = surge.Unmarshal(r, &prevote.Value, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling value: %v", err)
	}
	m, err = surge.Unmarshal(r, &prevote.From, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling from: %v", err)
	}
	m, err = surge.Unmarshal(r, &prevote.Signature, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling signature: %v", err)
	}
	return m, nil
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

func NewPrecommitHash(height Height, round Round, value Value) (id.Hash, error) {
	buf := new(bytes.Buffer)
	buf.Grow(surge.SizeHint(height) + surge.SizeHint(round) + surge.SizeHint(value))
	return NewPrecommitHashWithBuffer(height, round, value, buf)
}

func NewPrecommitHashWithBuffer(height Height, round Round, value Value, buf *bytes.Buffer) (id.Hash, error) {
	m, err := surge.Marshal(buf, height, surge.MaxBytes)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling height=%v: %v", height, err)
	}
	m, err = surge.Marshal(buf, round, m)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling round=%v: %v", round, err)
	}
	m, err = surge.Marshal(buf, value, m)
	if err != nil {
		return id.Hash{}, fmt.Errorf("marshaling value=%v: %v", value, err)
	}
	return id.NewHash(buf.Bytes()), nil
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
func (precommit Precommit) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, precommit.Height, m)
	if err != nil {
		return m, fmt.Errorf("marshaling height=%v: %v", precommit.Height, err)
	}
	m, err = surge.Marshal(w, precommit.Round, m)
	if err != nil {
		return m, fmt.Errorf("marshaling round=%v: %v", precommit.Round, err)
	}
	m, err = surge.Marshal(w, precommit.Value, m)
	if err != nil {
		return m, fmt.Errorf("marshaling value=%v: %v", precommit.Value, err)
	}
	m, err = surge.Marshal(w, precommit.From, m)
	if err != nil {
		return m, fmt.Errorf("marshaling from=%v: %v", precommit.From, err)
	}
	m, err = surge.Marshal(w, precommit.Signature, m)
	if err != nil {
		return m, fmt.Errorf("marshaling signature=%v: %v", precommit.Signature, err)
	}
	return m, nil
}

// Unmarshal binary into this message.
func (precommit *Precommit) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &precommit.Height, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling height: %v", err)
	}
	m, err = surge.Unmarshal(r, &precommit.Round, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling round: %v", err)
	}
	m, err = surge.Unmarshal(r, &precommit.Value, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling value: %v", err)
	}
	m, err = surge.Unmarshal(r, &precommit.From, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling from: %v", err)
	}
	m, err = surge.Unmarshal(r, &precommit.Signature, m)
	if err != nil {
		return m, fmt.Errorf("unmarshaling signature: %v", err)
	}
	return m, nil
}
