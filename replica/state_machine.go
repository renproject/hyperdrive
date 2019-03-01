package replica

import (
	"time"

	"github.com/renproject/hyperdrive/block"
)

type Dispatcher interface {
	Dispatch(action Action)
}

type StateMachine interface {
	Transition(state State, transition Transition) State
}

type stateMachine struct {
	dispatcher Dispatcher

	polkaBuilder       block.PolkaBuilder
	commitBuilder      block.CommitBuilder
	blockchain         block.Blockchain
	consensusThreshold int64
}

func NewStateMachine(dispatcher Dispatcher, polkaBuilder block.PolkaBuilder, commitBuilder block.CommitBuilder, blockchain block.Blockchain, consensusThreshold int64) StateMachine {
	return &stateMachine{
		dispatcher:         dispatcher,
		polkaBuilder:       polkaBuilder,
		commitBuilder:      commitBuilder,
		blockchain:         blockchain,
		consensusThreshold: consensusThreshold,
	}
}

func (stateMachine *stateMachine) Transition(state State, transition Transition) State {
	switch state := state.(type) {
	case WaitingForPropose:
		return stateMachine.waitForPropose(state, transition)

	case WaitingForPolka:
		return stateMachine.waitForPolka(state, transition)

	case WaitingForCommit:
		return stateMachine.waitForCommit(state, transition)
	}
	return nil
}

func (stateMachine *stateMachine) waitForPropose(state WaitingForPropose, transition Transition) State {
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
	return state
}

func (stateMachine *stateMachine) waitForPolka(state WaitingForPolka, transition Transition) State {
	switch transition := transition.(type) {
	case PreVoted:
		return stateMachine.reducePreVoted(state, transition)
	case PreCommitted:
		return stateMachine.reducePreCommitted(state, transition)
	}
	return state
}

func (stateMachine *stateMachine) waitForCommit(state WaitingForCommit, transition Transition) State {
	switch transition := transition.(type) {
	case PreCommitted:
		return stateMachine.reducePreCommitted(state, transition)
	}
	return state
}

func (stateMachine *stateMachine) reduceTimedOut(currentState State, timedOut TimedOut) State {
	stateMachine.dispatcher.Dispatch(PreVote{
		PreVote: block.PreVote{
			Block:  nil,
			Round:  currentState.Round(),
			Height: currentState.Height(),
		},
	})
	return WaitForPolka(currentState.Round(), currentState.Height())
}

func (stateMachine *stateMachine) reduceProposed(currentState State, proposed Proposed) State {
	if proposed.Block.Round != currentState.Round() {
		return currentState
	}
	if proposed.Block.Height != currentState.Height() {
		return currentState
	}
	if proposed.Block.Time.After(time.Now()) {
		return currentState
	}

	stateMachine.dispatcher.Dispatch(PreVote{
		PreVote: block.PreVote{
			Block:  &proposed.Block,
			Round:  proposed.Block.Round,
			Height: proposed.Block.Height,
		},
	})
	return WaitForPolka(currentState.Round(), currentState.Height())
}

func (stateMachine *stateMachine) reducePreVoted(currentState State, preVoted PreVoted) State {
	if preVoted.Round != currentState.Round() {
		return currentState
	}
	if preVoted.Height != currentState.Height() {
		return currentState
	}

	stateMachine.polkaBuilder.Insert(preVoted.SignedPreVote)
	if polka, ok := stateMachine.polkaBuilder.Polka(stateMachine.consensusThreshold); ok {
		stateMachine.dispatcher.Dispatch(PreCommit{
			PreCommit: block.PreCommit{
				Polka: polka,
			},
		})
		return WaitForCommit(polka)
	}

	stateMachine.dispatcher.Dispatch(PreVote{
		PreVote: block.PreVote{
			Block:  preVoted.Block,
			Round:  preVoted.Block.Round,
			Height: preVoted.Block.Height,
		},
	})
	return WaitForPolka(currentState.Round(), currentState.Height())
}

func (stateMachine *stateMachine) reducePreCommitted(currentState State, preCommitted PreCommitted) State {
	if preCommitted.Polka.Round != currentState.Round() {
		return currentState
	}
	if preCommitted.Polka.Height != currentState.Height() {
		return currentState
	}

	stateMachine.commitBuilder.Insert(preCommitted.SignedPreCommit)
	if commit, ok := stateMachine.commitBuilder.Commit(stateMachine.consensusThreshold); ok {
		if commit.Polka.Block == nil {
			return WaitForPropose(currentState.Round()+1, currentState.Height())
		}
		stateMachine.blockchain.Extend(commit)
		return WaitForPropose(currentState.Round(), currentState.Height()+1)
	}

	stateMachine.dispatcher.Dispatch(PreCommit{
		PreCommit: block.PreCommit{
			Polka: preCommitted.Polka,
		},
	})
	return WaitForCommit(preCommitted.Polka)
}
