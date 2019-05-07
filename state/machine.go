package state

import (
	"fmt"

	"github.com/renproject/hyperdrive/block"
)

type Machine interface {
	Transition(state State, transition Transition) (State, Action)
	Drop(height block.Height)
}

type machine struct {
	polkaBuilder       block.PolkaBuilder
	commitBuilder      block.CommitBuilder
	consensusThreshold int
}

func NewMachine(polkaBuilder block.PolkaBuilder, commitBuilder block.CommitBuilder, consensusThreshold int) Machine {
	return &machine{
		polkaBuilder:       polkaBuilder,
		commitBuilder:      commitBuilder,
		consensusThreshold: consensusThreshold,
	}
}

func (machine *machine) Transition(state State, transition Transition) (State, Action) {
	switch state := state.(type) {
	case WaitingForPropose:
		return machine.waitForPropose(state, transition)
	case WaitingForPolka:
		return machine.waitForPolka(state, transition)
	case WaitingForCommit:
		return machine.waitForCommit(state, transition)
	}
	return nil, nil
}

func (machine *machine) waitForPropose(state WaitingForPropose, transition Transition) (State, Action) {
	switch transition := transition.(type) {
	case TimedOut:
		return machine.reduceTimedOut(state, transition)
	case Proposed:
		return machine.reduceProposed(state, transition)
	case PreVoted:
		return machine.reducePreVoted(state, transition)
	case PreCommitted:
		return machine.reducePreCommitted(state, transition)
	}
	return state, nil
}

func (machine *machine) waitForPolka(state WaitingForPolka, transition Transition) (State, Action) {
	switch transition := transition.(type) {
	case PreVoted:
		return machine.reducePreVoted(state, transition)
	case PreCommitted:
		return machine.reducePreCommitted(state, transition)
	}
	return state, nil
}

func (machine *machine) waitForCommit(state WaitingForCommit, transition Transition) (State, Action) {
	switch transition := transition.(type) {
	case PreCommitted:
		return machine.reducePreCommitted(state, transition)
	}
	return state, nil
}

func (machine *machine) reduceTimedOut(currentState State, timedOut TimedOut) (State, Action) {
	return WaitForPolka(currentState.Round(), currentState.Height()), PreVote{
		PreVote: block.PreVote{
			Block:  nil,
			Round:  currentState.Round(),
			Height: currentState.Height(),
		},
	}
}

func (machine *machine) reduceProposed(currentState State, proposed Proposed) (State, Action) {
	if proposed.Block.Round != currentState.Round() {
		return currentState, nil
	}
	if proposed.Block.Height != currentState.Height() {
		return currentState, nil
	}

	return WaitForPolka(currentState.Round(), currentState.Height()), PreVote{
		PreVote: block.PreVote{
			Block:  &proposed.SignedBlock,
			Round:  proposed.SignedBlock.Round,
			Height: proposed.SignedBlock.Height,
		},
	}
}

func (machine *machine) reducePreVoted(currentState State, preVoted PreVoted) (State, Action) {
	if preVoted.Round < currentState.Round() {
		return currentState, nil
	}
	if preVoted.Height != currentState.Height() {
		return currentState, nil
	}

	if new := machine.polkaBuilder.Insert(preVoted.SignedPreVote); new {
		if polka, ok := machine.polkaBuilder.Polka(currentState.Height(), machine.consensusThreshold); ok {
			// Invariant check
			if polka.Round < currentState.Round() {
				panic(fmt.Errorf("expected polka round (%v) to be greater or equal to the current round (%v)", polka.Round, currentState.Round()))
			}

			return WaitForCommit(polka), PreCommit{
				PreCommit: block.PreCommit{
					Polka: polka,
				},
			}
		}
		return WaitForPolka(currentState.Round(), currentState.Height()), PreVote{
			PreVote: block.PreVote{
				Block:  preVoted.Block,
				Round:  preVoted.Round,
				Height: preVoted.Height,
			},
		}
	}
	return currentState, nil
}

func (machine *machine) reducePreCommitted(currentState State, preCommitted PreCommitted) (State, Action) {
	if preCommitted.Polka.Round < currentState.Round() {
		return currentState, nil
	}
	if preCommitted.Polka.Height != currentState.Height() {
		return currentState, nil
	}

	if new := machine.commitBuilder.Insert(preCommitted.SignedPreCommit); new {
		if commit, ok := machine.commitBuilder.Commit(currentState.Height(), machine.consensusThreshold); ok {
			// Invariant check
			if commit.Polka.Round < currentState.Round() {
				panic(fmt.Errorf("expected commit round (%v) to be greater or equal to the current round (%v)", commit.Polka.Round, currentState.Round()))
			}
			if commit.Polka.Block == nil {
				return WaitForPropose(commit.Polka.Round+1, currentState.Height()), Commit{
					Commit: commit,
				}
			}
			return WaitForPropose(commit.Polka.Round, currentState.Height()+1), Commit{
				Commit: commit,
			}
		}
		return WaitForCommit(preCommitted.Polka), PreCommit{
			PreCommit: block.PreCommit{
				Polka: preCommitted.Polka,
			},
		}
	}
	return currentState, nil
}

func (machine *machine) Drop(height block.Height) {
	machine.polkaBuilder.Drop(height)
	machine.commitBuilder.Drop(height)
}
