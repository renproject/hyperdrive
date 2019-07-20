package replica

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
)

// NewRoundRobinScheduler returns a `process.Scheduler` that implements a round
// robin schedule that weights the `block.Height` and the `block.Round` equally.
func NewRoundRobinScheduler(signatories block.Signatories) process.Scheduler {
	return &roundRobinScheduler{
		signatories: signatories,
	}
}

type roundRobinScheduler struct {
	signatories block.Signatories
}

func (scheduler *roundRobinScheduler) Schedule(height block.Height, round block.Round) block.Signatory {
	return scheduler.signatories[(uint64(height)+uint64(round))%uint64(len(scheduler.signatories))]
}
