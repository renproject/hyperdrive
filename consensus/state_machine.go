package consensus

import (
	"fmt"

	"github.com/renproject/hyperdrive/block"
)

type StateMachine interface {
	Transition(state State, transition Transition) (State, Action)
}

type stateMachine struct {
	polkaBuilder       block.PolkaBuilder
	commitBuilder      block.CommitBuilder
	consensusThreshold int
}

func NewStateMachine(polkaBuilder block.PolkaBuilder, commitBuilder block.CommitBuilder, consensusThreshold int) StateMachine {
	return &stateMachine{
		polkaBuilder:       polkaBuilder,
		commitBuilder:      commitBuilder,
		consensusThreshold: consensusThreshold,
	}
}

func (stateMachine *stateMachine) Transition(state State, transition Transition) (State, Action) {
	switch state := state.(type) {
	case WaitingForPropose:
		return stateMachine.waitForPropose(state, transition)
	case WaitingForPolka:
		return stateMachine.waitForPolka(state, transition)
	case WaitingForCommit:
		return stateMachine.waitForCommit(state, transition)
	}
	return nil, nil
}

func (stateMachine *stateMachine) waitForPropose(state WaitingForPropose, transition Transition) (State, Action) {
	switch transition := transition.(type) {
	case TimedOut:
		return stateMachine.reduceTimedOut(state, transition)
	case Proposed:
		return stateMachine.reduceProposed(state, transition)
	case PreVoted:
		return stateMachine.reducePreVoted(state, transition)
	case PreCommitted:
		return stateMachine.reducePreCommitted(state, transition)
	}
	return state, nil
}

func (stateMachine *stateMachine) waitForPolka(state WaitingForPolka, transition Transition) (State, Action) {
	switch transition := transition.(type) {
	case PreVoted:
		return stateMachine.reducePreVoted(state, transition)
	case PreCommitted:
		return stateMachine.reducePreCommitted(state, transition)
	}
	return state, nil
}

func (stateMachine *stateMachine) waitForCommit(state WaitingForCommit, transition Transition) (State, Action) {
	switch transition := transition.(type) {
	case PreCommitted:
		return stateMachine.reducePreCommitted(state, transition)
	}
	return state, nil
}

func (stateMachine *stateMachine) reduceTimedOut(currentState State, timedOut TimedOut) (State, Action) {
	return WaitForPolka(currentState.Round(), currentState.Height()), PreVote{
		PreVote: block.PreVote{
			Block:  nil,
			Round:  currentState.Round(),
			Height: currentState.Height(),
		},
	}
}

func (stateMachine *stateMachine) reduceProposed(currentState State, proposed Proposed) (State, Action) {
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

func (stateMachine *stateMachine) reducePreVoted(currentState State, preVoted PreVoted) (State, Action) {
	if preVoted.Round < currentState.Round() {
		return currentState, nil
	}
	if preVoted.Height != currentState.Height() {
		return currentState, nil
	}

	stateMachine.polkaBuilder.Insert(preVoted.SignedPreVote)
	if polka, ok := stateMachine.polkaBuilder.Polka(currentState.Height(), stateMachine.consensusThreshold); ok {
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

func (stateMachine *stateMachine) reducePreCommitted(currentState State, preCommitted PreCommitted) (State, Action) {
	if preCommitted.Polka.Round < currentState.Round() {
		return currentState, nil
	}
	if preCommitted.Polka.Height != currentState.Height() {
		return currentState, nil
	}

	stateMachine.commitBuilder.Insert(preCommitted.SignedPreCommit)
	if commit, ok := stateMachine.commitBuilder.Commit(currentState.Height(), stateMachine.consensusThreshold); ok {
		// Invariant check
		if commit.Polka.Round < currentState.Round() {
			panic(fmt.Errorf("expected commit round (%v) to be greater or equal to the current round (%v)", commit.Polka.Round, currentState.Round()))
		}
		if commit.Polka.Block == nil {
			return WaitForPropose(commit.Polka.Round+1, currentState.Height()), nil
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
