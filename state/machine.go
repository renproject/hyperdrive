package state

import (
	"fmt"

	"github.com/renproject/hyperdrive/block"
)

type Machine interface {
	Height() block.Height
	Round() block.Round
	State() State
	InsertPrevote(signedPreVote block.SignedPreVote)
	InsertPrecommit(signedPreCommit block.SignedPreCommit)
	SyncCommit(commit block.Commit)
	Drop()

	Transition(transition Transition) Action
}

type machine struct {
	index  int
	state  State
	height block.Height
	round  block.Round

	lockedRound *block.Round
	lockedBlock *block.SignedBlock
	lastAction  Action

	polkaBuilder       block.PolkaBuilder
	commitBuilder      block.CommitBuilder
	consensusThreshold int
}

func NewMachine(state State, polkaBuilder block.PolkaBuilder, commitBuilder block.CommitBuilder, consensusThreshold int) Machine {
	return &machine{
		index:              0,
		state:              state,
		polkaBuilder:       polkaBuilder,
		commitBuilder:      commitBuilder,
		consensusThreshold: consensusThreshold,
	}
}

func (machine *machine) Height() block.Height {
	return machine.height
}

func (machine *machine) Round() block.Round {
	return machine.round
}

func (machine *machine) State() State {
	return machine.state
}

func (machine *machine) InsertPrevote(prevote block.SignedPreVote) {
	machine.polkaBuilder.Insert(prevote)
}

func (machine *machine) InsertPrecommit(precommit block.SignedPreCommit) {
	machine.commitBuilder.Insert(precommit)
}

func (machine *machine) SyncCommit(commit block.Commit) {
	if commit.Polka.Height > machine.height {
		machine.state = WaitingForPropose{}
		machine.height = commit.Polka.Height + 1
		machine.round = 0
		machine.lockedBlock = nil
		machine.lockedRound = nil
	}
}

func (machine *machine) Drop() {
	if machine.index < 100 {
		fmt.Println("dropping everything at ", machine.height)
	}
	// machine.bufferredProposals = map[block.Round]block.SignedPropose{}
	machine.polkaBuilder.Drop(machine.height)
	machine.commitBuilder.Drop(machine.height)
}

func (machine *machine) Transition(transition Transition) Action {
	// Check pre-conditions
	if machine.lockedRound == nil {
		if machine.lockedBlock != nil {
			panic("expected locked block to be nil")
		}
	}
	if machine.lockedRound != nil {
		if machine.lockedBlock == nil {
			panic("expected locked round to be nil")
		}
	}

	switch machine.state.(type) {
	case WaitingForPropose:
		if machine.index < 100 {
			fmt.Printf("got %T while waiting for propose\n", transition)
		}
		return machine.waitForPropose(transition)
	case WaitingForPolka:
		if machine.index < 100 {
			fmt.Printf("got %T while waiting for polka\n", transition)
		}
		return machine.waitForPolka(transition)
	case WaitingForCommit:
		if machine.index < 100 {
			fmt.Printf("got %T while waiting for commit\n", transition)
		}
		return machine.waitForCommit(transition)
	default:
		panic(fmt.Errorf("unexpected state type %T", machine.state))
	}
}

func (machine *machine) waitForPropose(transition Transition) Action {
	switch transition := transition.(type) {
	case Proposed:
		// FIXME: Proposals can (optionally) include a Polka to encourage
		// unlocking faster than would otherwise be possible.

		if machine.index < 100 {
			fmt.Printf("changing to wait for polka at propose(H,R) = (%d, %d)\n", transition.Block.Height, transition.Round)
		}
		machine.state = WaitingForPolka{}
		machine.lastAction = machine.preVote(&transition.Block)
		return machine.lastAction

	case PreVoted:
		_ = machine.polkaBuilder.Insert(transition.SignedPreVote)

	case PreCommitted:
		_ = machine.commitBuilder.Insert(transition.SignedPreCommit)

	case TimedOut:
		if machine.index < 100 {
			fmt.Printf("changing to wait for polka at timedout\n")
		}
		machine.state = WaitingForPolka{}
		machine.lastAction = machine.preVote(nil)
		return machine.lastAction

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
	}

	return machine.checkCommonExitConditions()
}

func (machine *machine) waitForPolka(transition Transition) Action {
	switch transition := transition.(type) {
	case Proposed:
		// if transition.Round > machine.round {
		// 	machine.bufferredProposals[transition.Round] = transition.SignedPropose
		// }

	case PreVoted:
		if !machine.polkaBuilder.Insert(transition.SignedPreVote) {
			if machine.index < 100 {
				fmt.Println("not new")
			}
			return nil
		}

		// polka, polkaRound := machine.polkaBuilder.Polka(machine.height, machine.consensusThreshold)
		// if (polkaRound != nil && polka == nil) || (polka != nil && polka.Round == machine.round) {
		// 	if machine.index < 100 { // fmt.Printf("changing to wait for commit on receiving polka (H,R) = (%d, %d) for prevote\n", polka.Height, polka.Round)
		// 	}
		polka, _ := machine.polkaBuilder.Polka(machine.height, machine.consensusThreshold)
		if polka != nil && polka.Round == machine.round {
			fmt.Printf("changing to wait for commit on receiving polka (H,R) = (%d, %d) for prevote\n", polka.Height, polka.Round)
			machine.state = WaitingForCommit{}
			machine.lastAction = machine.preCommit()
			return machine.lastAction
		}

	case PreCommitted:
		if !machine.commitBuilder.Insert(transition.SignedPreCommit) {
			return nil
		}

	case TimedOut:
		_, preVotingRound := machine.polkaBuilder.Polka(machine.height, machine.consensusThreshold)
		if preVotingRound == nil {
			return machine.lastAction
		}

		if machine.index < 100 {
			fmt.Printf("changing to wait for commit on receiving timeout\n")
		}
		machine.state = WaitingForCommit{}
		machine.lastAction = machine.preCommit()
		return machine.lastAction

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
	}

	return machine.checkCommonExitConditions()
}

func (machine *machine) waitForCommit(transition Transition) Action {
	switch transition := transition.(type) {
	case Proposed:
		// if transition.Round > machine.round {
		// 	machine.bufferredProposals[transition.Round] = transition.SignedPropose
		// }

	case PreVoted:
		_ = machine.polkaBuilder.Insert(transition.SignedPreVote)

	case PreCommitted:
		if !machine.commitBuilder.Insert(transition.SignedPreCommit) {
			return nil
		}

		// commit, commitRound := machine.commitBuilder.Commit(machine.height, machine.consensusThreshold)
		// if (commitRound != nil && commit == nil) || (commit != nil && commit.Polka.Block == nil && commit.Polka.Round == machine.round) {
		// 	if machine.index < 100 {
		// 		fmt.Printf("changing to wait for propose on receiving commit (H,R) = (%d, %d) for precommit (H,R) = (%d, %d)\n", commit.Polka.Height, commit.Polka.Round, transition.Polka.Height, transition.Polka.Round)
		// 	}
		commit, _ := machine.commitBuilder.Commit(machine.height, machine.consensusThreshold)
		if commit != nil && commit.Polka.Block == nil && commit.Polka.Round == machine.round {
			fmt.Printf("changing to wait for propose on receiving commit (H,R) = (%d, %d) for precommit (H,R) = (%d, %d)\n", commit.Polka.Height, commit.Polka.Round, transition.Polka.Height, transition.Polka.Round)
			machine.state = WaitingForPropose{}
			machine.lastAction = nil
			machine.round++
			return Commit{
				Commit: block.Commit{
					Polka: block.Polka{
						Height: machine.height,
						Round:  machine.round,
					},
				},
			}
		}

	case TimedOut:
		_, preCommittingRound := machine.commitBuilder.Commit(machine.height, machine.consensusThreshold)
		if preCommittingRound == nil {
			return machine.lastAction
		}

		if machine.index < 100 {
			fmt.Printf("changing to wait for propose on receiving timeout\n")
		}
		machine.state = WaitingForPropose{}
		machine.lastAction = nil
		machine.round++
		return Commit{
			Commit: block.Commit{
				Polka: block.Polka{
					Height: machine.height,
					Round:  machine.round,
				},
			},
		}

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
	}

	return machine.checkCommonExitConditions()
}

func (machine *machine) preVote(proposedBlock *block.SignedBlock) Action {
	polka, _ := machine.polkaBuilder.Polka(machine.height, machine.consensusThreshold)

	if machine.lockedRound != nil && polka != nil {
		// If the validator is locked on a block since LastLockRound but now has
		// a PoLC for something else at round PoLC-Round where LastLockRound <
		// PoLC-Round < R, then it unlocks.
		if *machine.lockedRound < polka.Round {
			machine.lockedRound = nil
			machine.lockedBlock = nil
		}
	}

	if machine.lockedRound != nil {
		// If the validator is still locked on a block, it prevotes that.
		return PreVote{
			PreVote: block.PreVote{
				Block:  machine.lockedBlock,
				Height: machine.height,
				Round:  machine.round,
			},
		}
	}

	if proposedBlock != nil && proposedBlock.Height == machine.height {
		// Else, if the proposed block from Propose(H,R) is good, it prevotes that.
		return PreVote{
			PreVote: block.PreVote{
				Block:  proposedBlock,
				Height: machine.height,
				Round:  machine.round,
			},
		}
	}

	// Else, if the proposal is invalid or wasn't received on time, it prevotes <nil>.
	return PreVote{
		PreVote: block.PreVote{
			Block:  nil,
			Height: machine.height,
			Round:  machine.round,
		},
	}
}

func (machine *machine) preCommit() Action {
	polka, _ := machine.polkaBuilder.Polka(machine.height, machine.consensusThreshold)

	if polka != nil {
		if polka.Block != nil {
			// If the validator has a PoLC at (H,R) for a particular block B, it
			// (re)locks (or changes lock to) and precommits B and sets LastLockRound =
			// R.
			machine.lockedRound = &polka.Round
			machine.lockedBlock = polka.Block
			return PreCommit{
				PreCommit: block.PreCommit{
					Polka: *polka,
				},
			}
		}

		// Else, if the validator has a PoLC at (H,R) for <nil>, it unlocks and
		// precommits <nil>.
		machine.lockedRound = nil
		machine.lockedBlock = nil
		return PreCommit{
			PreCommit: block.PreCommit{
				Polka: *polka,
			},
		}
	}

	// Else, it keeps the lock unchanged and precommits <nil>.
	return PreCommit{
		PreCommit: block.PreCommit{
			Polka: block.Polka{
				Height: machine.height,
				Round:  machine.round,
			},
		},
	}
}

func (machine *machine) checkCommonExitConditions() Action {
	// Get the Commit for the current Height and the latest Round
	commit, preCommittingRound := machine.commitBuilder.Commit(machine.height, machine.consensusThreshold)
	if commit != nil && commit.Polka.Block != nil {
		// After +2/3 precommits for a particular block. --> goto Commit(H)
		if machine.index < 100 {
			fmt.Printf("changing to wait for propose on receiving commit (H,R) = (%d, %d)\n", commit.Polka.Height, commit.Polka.Round)
		}
		machine.state = WaitingForPropose{}
		machine.height = commit.Polka.Height + 1
		machine.round = 0
		machine.lastAction = nil
		machine.lockedBlock = nil
		machine.lockedRound = nil
		return Commit{Commit: *commit}
	}

	// Get the Polka for the current Height and the latest Round
	_, preVotingRound := machine.polkaBuilder.Polka(machine.height, machine.consensusThreshold)
	if preVotingRound != nil && *preVotingRound > machine.round {
		// After any +2/3 prevotes received at (H,R+x). --> goto Prevote(H,R+x)
		machine.round = *preVotingRound
		if machine.index < 100 {
			fmt.Printf("prevoting nil at stage %T \n", machine.state)
		}
		// machine.state = WaitingForPolka{}
		machine.lastAction = machine.preVote(nil)
		return machine.lastAction
	}

	if preCommittingRound != nil && *preCommittingRound > machine.round {
		// After any +2/3 precommits received at (H,R+x). --> goto Precommit(H,R+x)
		if machine.index < 100 {
			fmt.Printf("changing to wait for commit on receiving 2/3+ commits\n")
		}
		if machine.index < 100 {
			fmt.Printf("precommiting nil at stage %T \n", machine.state)
		}
		machine.state = WaitingForCommit{}
		machine.round = *preCommittingRound
		machine.lastAction = machine.preCommit()
		return machine.lastAction
	}

	return nil
}
