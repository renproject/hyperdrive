// Package schedule defines interfaces and implementations for scheduling
// different Processes as Block proposers. At any given Height and Round,
// exactly one Process is expected to take responsibility for proposing a Block,
// and this is determined by the Scheduler.
//
// It is important that all Processes agree on the schedule. That is, at any
// given Height and Round, all Processes must arrive at the same decision
// regarding which Process is expected to be the proposer. This is most commonly
// done by making the schedule deterministic based and locally computable. This
// means that Processes do not have to invoke a consensus algorithm in order to
// agree on the schedule (although, this is possible to do, using Block N to
// agree on the schedule for Block N+1).
package schedule

import (
	"sync"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
)

// A Scheduler is used to determine which Signatory is expected to propose a
// Block at any given Height and Round. It supports rebasing, allowing the
// underlying Signatories to be changed over time. Schedulers are expected to be
// safe for concurrent use.
type Scheduler interface {
	// Schedule returns the Signatory that is expected to propose a Block at the
	// given Height and Round. If no Signatory is expected to propose a Block,
	// then it returns the InvalidSignatory.
	//
	// Schedule must return the same Signatory for the same Height and Round for
	// all consensus participants. For performance, it should be locally
	// computable; there should be no communication between consensus
	// participants.
	//
	//  proposer := scheduler.Schedule(propose.Height(), propose.Round())
	//  Expect(propose.Signatory()).To(Equal(proposer))
	//
	Schedule(block.Height, block.Round) id.Signatory

	// Rebase tells the Scheduler that the underlying Signatories has been
	// changed. The Scheduler must only ever schedule Signatories from that most
	// recent rebase.
	//
	//  if block.Kind() == Rebase {
	//      scheduler.Rebase(block.Header().Signatories())
	//  }
	//
	Rebase(id.Signatories)
}

type roundRobin struct {
	signatoriesMu *sync.Mutex
	signatories   id.Signatories
}

// RoundRobin returns a Scheduler that uses a round-robin scheduling algorithm
// to select a Signatory. Round-robin scheduling has the advantage of being very
// easy to implement and understand, but has the disadvantage of being unfair.
// As such, it should be avoided when the proposer is expected to receive a
// larger Block reward than non-proposers.
func RoundRobin(signatories id.Signatories) Scheduler {
	rr := &roundRobin{
		signatoriesMu: new(sync.Mutex),
	}
	rr.Rebase(signatories)
	return rr
}

// Schedule a Signatory using the sum of the Height and Round, modulo the number
// of Signatories, to select a Signatory.
func (rr *roundRobin) Schedule(height block.Height, round block.Round) id.Signatory {
	rr.signatoriesMu.Lock()
	defer rr.signatoriesMu.Unlock()

	if len(rr.signatories) == 0 {
		// When there are no signatories, there is no valid Signatory to select.
		// As per the requirements for the Scheduler interface, we return the
		// InvalidSignatory.
		return block.InvalidSignatory
	}
	if height == block.InvalidHeight {
		// When the height is invalid, there is no valid Signatory to select.
		return block.InvalidSignatory
	}
	if round == block.InvalidRound {
		// When the round is invalid, there is no valid Signatory to select.
		return block.InvalidSignatory
	}

	return rr.signatories[(uint64(height)+uint64(round))%uint64(len(rr.signatories))]
}

// Rebase will replace the underlying Signatories from which the round-robin
// Scheduler will select proposers.
func (rr *roundRobin) Rebase(signatories id.Signatories) {
	rr.signatoriesMu.Lock()
	defer rr.signatoriesMu.Unlock()

	// Copy signatories into the scheduler to avoid manipulation of the slice,
	// external to the scheduler, from affecting the scheduler.
	rr.signatories = make(id.Signatories, len(signatories))
	copy(rr.signatories, signatories)
}
