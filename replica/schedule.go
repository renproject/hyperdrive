package replica

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/id"
)

type roundRobinScheduler struct {
	signatories id.Signatories
}

// newRoundRobinScheduler returns a `process.Scheduler` that implements a round
// robin schedule that weights the `block.Height` and the `block.Round` equally.
func newRoundRobinScheduler(signatories id.Signatories) *roundRobinScheduler {
	return &roundRobinScheduler{
		signatories: signatories,
	}
}

func (scheduler *roundRobinScheduler) Schedule(height block.Height, round block.Round) id.Signatory {
	return scheduler.signatories[(uint64(height)+uint64(round))%uint64(len(scheduler.signatories))]
}

func (scheduler *roundRobinScheduler) rebase(sigs id.Signatories) {
	scheduler.signatories = sigs
}
