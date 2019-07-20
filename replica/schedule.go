package replica

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/id"
	"github.com/renproject/hyperdrive/process"
)

type roundRobinScheduler struct {
	signatories id.Signatories
}

// newRoundRobinScheduler returns a `process.Scheduler` that implements a round
// robin schedule that weights the `block.Height` and the `block.Round` equally.
func newRoundRobinScheduler(signatories id.Signatories) process.Scheduler {
	return &roundRobinScheduler{
		signatories: signatories,
	}
}

func (scheduler *roundRobinScheduler) Schedule(height block.Height, round block.Round) id.Signatory {
	return scheduler.signatories[(uint64(height)+uint64(round))%uint64(len(scheduler.signatories))]
}
