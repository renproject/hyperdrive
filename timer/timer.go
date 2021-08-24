package timer

import (
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/process"

	"github.com/renproject/surge"
)

// Timeout represents an event emitted by the Linear Timer whenever
// a scheduled timeout is triggered
type Timeout struct {
	MessageType process.MessageType
	Height      process.Height
	Round       process.Round
}

// SizeHint implements surge SizeHinter for Timeout
func (timeout Timeout) SizeHint() int {
	return surge.SizeHintI8 +
		surge.SizeHint(timeout.Height) +
		surge.SizeHint(timeout.Round)
}

// Marshal implements surge Marshaler for Timeout
func (timeout Timeout) Marshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.MarshalI8(int8(timeout.MessageType), buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling MeesageType=%v: %v", timeout.MessageType, err)
	}
	buf, rem, err = surge.Marshal(timeout.Height, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling Height=%v: %v", timeout.Height, err)
	}
	buf, rem, err = surge.Marshal(timeout.Round, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling Round=%v: %v", timeout.Round, err)
	}

	return buf, rem, nil
}

// Unmarshal implements surge Unmarshaler for Timeout
func (timeout *Timeout) Unmarshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.UnmarshalI8((*int8)(&timeout.MessageType), buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling MessageType: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&timeout.Height, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling Height: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&timeout.Round, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling Round: %v", err)
	}

	return buf, rem, nil
}

// LinearTimer defines a timer that implements a timing out functionality.
// The timeouts for different contexts (Propose, Prevote and Precommit) are
// provided as callback functions that handle the corresponding timeouts. The
// timeout scales linearly with the consensus round
type LinearTimer struct {
	opts                   Options
	handleTimeoutPropose   func(Timeout)
	handleTimeoutPrevote   func(Timeout)
	handleTimeoutPrecommit func(Timeout)
}

// NewLinearTimer constructs a new Linear Timer from the input options and channels
func NewLinearTimer(opts Options, handleTimeoutPropose, handleTimeoutPrevote, handleTimeoutPrecommit func(Timeout)) *LinearTimer {
	return &LinearTimer{
		opts:                   opts,
		handleTimeoutPropose:   handleTimeoutPropose,
		handleTimeoutPrevote:   handleTimeoutPrevote,
		handleTimeoutPrecommit: handleTimeoutPrecommit,
	}
}

// TimeoutPropose schedules a propose timeout with a timeout period appropriately
// calculated for the consensus height and round
func (t *LinearTimer) TimeoutPropose(height process.Height, round process.Round) {
	if t.handleTimeoutPropose != nil {
		go func() {
			time.Sleep(t.DurationAtHeightAndRound(height, round))
			t.handleTimeoutPropose(Timeout{MessageType: process.MessageTypePropose, Height: height, Round: round})
		}()
	}
}

// TimeoutPrevote schedules a prevote timeout with a timeout period appropriately
// calculated for the consensus height and round
func (t *LinearTimer) TimeoutPrevote(height process.Height, round process.Round) {
	if t.handleTimeoutPrevote != nil {
		go func() {
			time.Sleep(t.DurationAtHeightAndRound(height, round))
			t.handleTimeoutPrevote(Timeout{MessageType: process.MessageTypePrevote, Height: height, Round: round})
		}()
	}
}

// TimeoutPrecommit schedules a precommit timeout with a timeout period appropriately
// calculated for the consensus height and round
func (t *LinearTimer) TimeoutPrecommit(height process.Height, round process.Round) {
	if t.handleTimeoutPrecommit != nil {
		go func() {
			time.Sleep(t.DurationAtHeightAndRound(height, round))
			t.handleTimeoutPrecommit(Timeout{MessageType: process.MessageTypePrecommit, Height: height, Round: round})
		}()
	}
}

// DurationAtHeightAndRound returns the duration of the timeout at the given
// height and round. This is the duration that the other methods will wait
// before scheduling their respective timeout events.
func (t *LinearTimer) DurationAtHeightAndRound(height process.Height, round process.Round) time.Duration {
	return t.opts.Timeout + t.opts.Timeout*time.Duration(float64(round)*t.opts.TimeoutScaling)
}
