package replica

import (
	"math"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
)

type backOffTimer struct {
	exp  float64
	base time.Duration
	max  time.Duration
}

func newBackOffTimer(exp float64, base time.Duration, max time.Duration) process.Timer {
	return &backOffTimer{
		exp:  exp,
		base: base,
		max:  max,
	}
}

func (timer *backOffTimer) Timeout(step process.Step, round block.Round) time.Duration {
	if round == 0 {
		return timer.base
	}
	multiplier := math.Pow(timer.exp, float64(round))
	var duration time.Duration

	// Make sure it doesn't overflow
	durationFloat := float64(timer.base) * multiplier
	if durationFloat > math.MaxInt64 {
		duration = time.Duration(math.MaxInt64)
	} else {
		duration = time.Duration(durationFloat)
	}

	if duration > timer.max {
		return timer.max
	}
	return duration
}
