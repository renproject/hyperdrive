// Package replica core package
//
// I assume that all provided `Transition`s are well formed and valid by
// the time they reach this module.
package replica

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/supervisor"
)

type Dispatcher interface {
	Dispatch(action consensus.Action)
}

type replica struct {
	transitions      <-chan consensus.Transition
	transitionBuffer consensus.TransitionBuffer
	dispatcher       Dispatcher
	state            consensus.State
	stateMachine     consensus.StateMachine
	blockchain       block.Blockchain
}

func New(
	transitions <-chan consensus.Transition,
	transitionBuffer consensus.TransitionBuffer,
	dispatcher Dispatcher,
	state consensus.State,
	stateMachine consensus.StateMachine,
	blockchain block.Blockchain,
) supervisor.Runner {
	return &replica{
		transitions:      transitions,
		transitionBuffer: transitionBuffer,
		dispatcher:       dispatcher,
		state:            state,
		stateMachine:     stateMachine,
		blockchain:       blockchain,
	}
}

func (replica *replica) Run(ctx supervisor.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case transition := <-replica.transitions:
			if replica.shouldDropTransition(transition) {
				continue
			}
			if replica.shouldBufferTransition(transition) {
				replica.transitionBuffer.Enqueue(transition)
				continue
			}

			for ok := true; ok; transition, ok = replica.transitionBuffer.Dequeue(replica.state.Height()) {
				nextState, action := replica.stateMachine.Transition(replica.state, transition)
				replica.dispatchAction(action)
				replica.state = nextState
				replica.transitionBuffer.Drop(replica.state.Height())
			}
		}
	}
}

func (replica *replica) dispatchAction(action consensus.Action) {
	if action == nil {
		return
	}
	switch action := action.(type) {
	case consensus.PreVote:
		replica.handlePreVote(action)
	case consensus.PreCommit:
		replica.handlePreCommit(action)
	case consensus.Commit:
		replica.handleCommit(action)
	}
	replica.dispatcher.Dispatch(action)
}

func (replica *replica) handlePreVote(preVote consensus.PreVote) {
}

func (replica *replica) handlePreCommit(preCommit consensus.PreCommit) {
}

func (replica *replica) handleCommit(commit consensus.Commit) {
	replica.blockchain.Extend(commit.Commit)
}

func (replica *replica) shouldDropTransition(transition consensus.Transition) bool {
	switch transition := transition.(type) {
	case consensus.Proposed:
		if transition.Height < replica.state.Height() {
			return true
		}
	case consensus.PreVoted:
		if transition.Height < replica.state.Height() {
			return true
		}
	case consensus.PreCommitted:
		if transition.Polka.Height < replica.state.Height() {
			return true
		}
	}
	return false
}

func (replica *replica) shouldBufferTransition(transition consensus.Transition) bool {
	switch transition := transition.(type) {
	case consensus.Proposed:
		if transition.Height > replica.state.Height() {
			return true
		}
	case consensus.PreVoted:
		if transition.Height > replica.state.Height() {
			return true
		}
	case consensus.PreCommitted:
		if transition.Polka.Height > replica.state.Height() {
			return true
		}
	}
	return false
}
