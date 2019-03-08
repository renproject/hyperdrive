package replica

import (
	"time"

	"github.com/renproject/hyperdrive/block"
)

type StateMachine interface {
	Transition(state State, transition Transition) (State, Action)
}

type stateMachine struct {
	polkaBuilder       block.PolkaBuilder
	commitBuilder      block.CommitBuilder
	consensusThreshold int64
}

func NewStateMachine(polkaBuilder block.PolkaBuilder,
	commitBuilder block.CommitBuilder,
	consensusThreshold int64) StateMachine {
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
	if proposed.Block.Time.After(time.Now()) {
		return currentState, nil
	}

	return WaitForPolka(currentState.Round(), currentState.Height()), PreVote{
		PreVote: block.PreVote{
			Block:  &proposed.Block,
			Round:  proposed.Block.Round,
			Height: proposed.Block.Height,
		},
	}
}

func (stateMachine *stateMachine) reducePreVoted(currentState State, preVoted PreVoted) (State, Action) {
	if preVoted.Round != currentState.Round() {
		return currentState, nil
	}
	if preVoted.Height != currentState.Height() {
		return currentState, nil
	}

	stateMachine.polkaBuilder.Insert(preVoted.SignedPreVote)
	if polka, ok := stateMachine.polkaBuilder.Polka(stateMachine.consensusThreshold); ok {
		return WaitForCommit(polka), PreCommit{
			PreCommit: block.PreCommit{
				Polka: polka,
			},
		}
	}

	return WaitForPolka(currentState.Round(), currentState.Height()), PreVote{
		PreVote: block.PreVote{
			Block:  preVoted.Block,
			Round:  preVoted.Block.Round,
			Height: preVoted.Block.Height,
		},
	}
}

func (stateMachine *stateMachine) reducePreCommitted(currentState State, preCommitted PreCommitted) (State, Action) {
	if preCommitted.Polka.Round != currentState.Round() {
		return currentState, nil
	}
	if preCommitted.Polka.Height != currentState.Height() {
		return currentState, nil
	}

	stateMachine.commitBuilder.Insert(preCommitted.SignedPreCommit)
	if commit, ok := stateMachine.commitBuilder.Commit(stateMachine.consensusThreshold); ok {
		if commit.Polka.Block == nil {
			return WaitForPropose(currentState.Round()+1, currentState.Height()), nil
		}
		return WaitForPropose(currentState.Round(), currentState.Height()+1), Commit{
			Commit: commit,
		}
	}

	return WaitForCommit(preCommitted.Polka), PreCommit{
		PreCommit: block.PreCommit{
			Polka: preCommitted.Polka,
		},
	}
}
