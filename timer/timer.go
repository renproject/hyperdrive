package timer

import (
	"time"

	"github.com/renproject/hyperdrive/process"
)

// Timeout represents an event emitted by the Linear Timer whenever
// a scheduled timeout is triggered
type Timeout struct {
	Height process.Height
	Round  process.Round
}

// LinearTimer defines a timer that implements a timing out functionality.
// The timeouts for different contexts (Propose, Prevote and Precommit) are
// emitted via separate channels. The timeout scales linearly with the
// consensus round
type LinearTimer struct {
	opts               Options
	onTimeoutPropose   chan<- Timeout
	onTimeoutPrevote   chan<- Timeout
	onTimeoutPrecommit chan<- Timeout
}

// NewLinearTimer constructs a new Linear Timer from the input options and channels
func NewLinearTimer(opts Options, onTimeoutPropose, onTimeoutPrevote, onTimeoutPrecommit chan<- Timeout) process.Timer {
	return &LinearTimer{
		opts:               opts,
		onTimeoutPropose:   onTimeoutPropose,
		onTimeoutPrevote:   onTimeoutPrevote,
		onTimeoutPrecommit: onTimeoutPrecommit,
	}
}

// TimeoutPropose schedules a propose timeout with a timeout period appropriately
// calculated for the consensus height and round
func (t *LinearTimer) TimeoutPropose(height process.Height, round process.Round) {
	go func() {
		time.Sleep(t.timeoutDuration(height, round))
		t.onTimeoutPropose <- Timeout{Height: height, Round: round}
	}()
}

// TimeoutPrevote schedules a prevote timeout with a timeout period appropriately
// calculated for the consensus height and round
func (t *LinearTimer) TimeoutPrevote(height process.Height, round process.Round) {
	go func() {
		time.Sleep(t.timeoutDuration(height, round))
		t.onTimeoutPrevote <- Timeout{Height: height, Round: round}
	}()
}

// TimeoutPrecommit schedules a precommit timeout with a timeout period appropriately
// calculated for the consensus height and round
func (t *LinearTimer) TimeoutPrecommit(height process.Height, round process.Round) {
	go func() {
		time.Sleep(t.timeoutDuration(height, round))
		t.onTimeoutPrecommit <- Timeout{Height: height, Round: round}
	}()
}

func (t *LinearTimer) timeoutDuration(height process.Height, round process.Round) time.Duration {
	return t.opts.Timeout + t.opts.Timeout*time.Duration(float64(round)*t.opts.TimeoutScaling)
}
