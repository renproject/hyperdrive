package timer

import (
	"time"

	"github.com/renproject/hyperdrive/process"
)

type Timeout struct {
	Height process.Height
	Round  process.Round
}

type LinearTimer struct {
	opts               Options
	onTimeoutPropose   chan<- Timeout
	onTimeoutPrevote   chan<- Timeout
	onTimeoutPrecommit chan<- Timeout
}

func NewLinearTimer(opts Options, onTimeoutPropose, onTimeoutPrevote, onTimeoutPrecommit chan<- Timeout) process.Timer {
	return &LinearTimer{
		opts:               opts,
		onTimeoutPropose:   onTimeoutPropose,
		onTimeoutPrevote:   onTimeoutPrevote,
		onTimeoutPrecommit: onTimeoutPrecommit,
	}
}

func (t *LinearTimer) TimeoutPropose(height process.Height, round process.Round) {
	go func() {
		time.Sleep(t.timeoutDuration(height, round))
		t.onTimeoutPropose <- Timeout{Height: height, Round: round}
	}()
}

func (t *LinearTimer) TimeoutPrevote(height process.Height, round process.Round) {
	go func() {
		time.Sleep(t.timeoutDuration(height, round))
		t.onTimeoutPrevote <- Timeout{Height: height, Round: round}
	}()
}

func (t *LinearTimer) TimeoutPrecommit(height process.Height, round process.Round) {
	go func() {
		time.Sleep(t.timeoutDuration(height, round))
		t.onTimeoutPrecommit <- Timeout{Height: height, Round: round}
	}()
}

func (t *LinearTimer) timeoutDuration(height process.Height, round process.Round) time.Duration {
	return t.opts.Timeout + t.opts.Timeout*time.Duration(float64(round)*t.opts.TimeoutScaling)
}
