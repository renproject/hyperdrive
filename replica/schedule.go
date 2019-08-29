package replica

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

type roundRobinScheduler struct {
	signatories id.Signatories
}

// NewRoundRobinScheduler returns a `process.Scheduler` that implements a round
// robin schedule that weights the `block.Height` and the `block.Round` equally.
func NewRoundRobinScheduler(signatories id.Signatories) process.Scheduler {
	return &roundRobinScheduler{
		// FIXME: Add a private `rebaseToNewSigs` method to allow the scheduler
		// to work with a new sig set.
		signatories: signatories,
	}
}

func (scheduler *roundRobinScheduler) Schedule(height block.Height, round block.Round) id.Signatory {
	return scheduler.signatories[(uint64(height)+uint64(round))%uint64(len(scheduler.signatories))]
}

// func(scheduler *roundRobinScheduler) rebaseToNewSigs(signatories id.Signatories) {
// 	scheduler.signatories = signatories
// }
