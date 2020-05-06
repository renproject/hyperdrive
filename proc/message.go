package proc

import (
	"fmt"
	"io"

	"github.com/renproject/surge"
)

type Propose struct {
	Height     Height `json:"height"`
	Round      Round  `json:"round"`
	ValidRound Round  `json:"validRound"`
	Value      Value  `json:"value"`
	From       Pid    `json:"from"`
}

func (propose *Propose) Equal(other *Propose) bool {
	return propose.Height == other.Height &&
		propose.Round == other.Round &&
		propose.ValidRound == other.ValidRound &&
		propose.Value.Equal(&other.Value) &&
		propose.From.Equal(&other.From)
}

func (propose *Propose) SizeHint() int {
	return surge.SizeHint(propose.Height) +
		surge.SizeHint(propose.Round) +
		surge.SizeHint(propose.ValidRound) +
		surge.SizeHint(propose.Value) +
		surge.SizeHint(propose.From)
}

func (propose *Propose) Marshal(w io.Writer, m int) (int, error) {
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
	return m, nil
}

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
	return m, nil
}

type Prevote struct {
	Height Height `json:"height"`
	Round  Round  `json:"round"`
	Value  Value  `json:"value"`
	From   Pid    `json:"from"`
}

func (prevote *Prevote) Equal(other *Prevote) bool {
	return prevote.Height == other.Height &&
		prevote.Round == other.Round &&
		prevote.Value.Equal(&other.Value) &&
		prevote.From.Equal(&other.From)
}

func (prevote *Prevote) SizeHint() int {
	return surge.SizeHint(prevote.Height) +
		surge.SizeHint(prevote.Round) +
		surge.SizeHint(prevote.Value) +
		surge.SizeHint(prevote.From)
}

func (prevote *Prevote) Marshal(w io.Writer, m int) (int, error) {
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
	return m, nil
}

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
	return m, nil
}

type Precommit struct {
	Height Height `json:"height"`
	Round  Round  `json:"round"`
	Value  Value  `json:"value"`
	From   Pid    `json:"from"`
}

func (precommit *Precommit) Equal(other *Precommit) bool {
	return precommit.Height == other.Height &&
		precommit.Round == other.Round &&
		precommit.Value.Equal(&other.Value) &&
		precommit.From.Equal(&other.From)
}

func (precommit *Precommit) SizeHint() int {
	return surge.SizeHint(precommit.Height) +
		surge.SizeHint(precommit.Round) +
		surge.SizeHint(precommit.Value) +
		surge.SizeHint(precommit.From)
}

func (precommit *Precommit) Marshal(w io.Writer, m int) (int, error) {
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
	return m, nil
}

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
	return m, nil
}
