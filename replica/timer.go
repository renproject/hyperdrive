package replica

import (
	"math"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/block"
)

func BackOffTimer(exp float64, base time.Duration, max time.Duration) process.Timer {
	return &backOffTimer{
		exp: exp,
		max: max,
	}
}

type backOffTimer struct {
	exp  float64
	base time.Duration
	max  time.Duration
}

func (timer *backOffTimer) Timeout(step process.Step, round block.Round) time.Duration {
	if round == 0 {
		return timer.base
	}
	multiplier := math.Pow(timer.exp, float64(round))
	duration := time.Duration(float64(timer.base) * multiplier)
	if duration > timer.max {
		return timer.max
	}
	return duration
}
