// Package scheduler defines interfaces and implementations for scheduling
// different processes as value proposers. At any given height and round,
// exactly one process is expected to take responsibility for proposing a value,
// and this is determined by the Scheduler.
//
// It is important that all processes agree on the schedule. That is, at any
// given height and round, all processes must arrive at the same decision
// regarding which process is expected to be the proposer. This is most commonly
// done by making the schedule deterministic and locally computable. This means
// that processes do not have to invoke a consensus algorithm in order to agree
// on the schedule (although, this is possible to do, using values from height N
// to agree on the schedule for height N+1).
package scheduler

import (
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

// RoundRobin holds a list of signatories that will participate in the round
// robin scheduling
type RoundRobin struct {
	signatories []id.Signatory
}

// NewRoundRobin returns a Scheduler that uses a simple round-robin scheduling
// algorithm to select a proposer. Round-robin scheduling has the advantage of
// being very easy to implement and understand, but has the disadvantage of
// being unfair. As such, it should be avoided when the proposer is expected to
// receive a reward.
func NewRoundRobin(signatories []id.Signatory) process.Scheduler {
	copied := make([]id.Signatory, len(signatories))
	copy(copied[:], signatories)
	return &RoundRobin{
		signatories: copied,
	}
}

// Schedule a proposer using the sum of the height and round, modulo the number
// of candidate processes, as an index into the current slice of candidate
// processes.
func (rr *RoundRobin) Schedule(height process.Height, round process.Round) id.Signatory {
	if len(rr.signatories) == 0 {
		panic("no processes to schedule")
	}
	if height <= 0 {
		panic("invalid height")
	}
	if round <= process.InvalidRound {
		panic("invalid round")
	}
	return rr.signatories[(uint64(height)+uint64(round))%uint64(len(rr.signatories))]
}
