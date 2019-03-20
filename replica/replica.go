package replica

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/supervisor"
)

type Dispatcher interface {
	Dispatch(action Action)
}

type replica struct {
	transitions      <-chan Transition
	transitionBuffer TransitionBuffer
	dispatcher       Dispatcher
	state            State
	stateMachine     StateMachine
	blockchain       block.Blockchain
}

func New(transitions <-chan Transition,
	transitionBuffer TransitionBuffer,
	dispatcher Dispatcher,
	state State,
	stateMachine StateMachine,
	blockchain block.Blockchain) supervisor.Runner {
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

func (replica *replica) dispatchAction(action Action) {
	if action == nil {
		return
	}
	switch action := action.(type) {
	case PreVote:
		replica.handlePreVote(action)
	case PreCommit:
		replica.handlePreCommit(action)
	case Commit:
		replica.handleCommit(action)
	}
	replica.dispatcher.Dispatch(action)
}

func (replica *replica) handlePreVote(preVote PreVote) {
}

func (replica *replica) handlePreCommit(preCommit PreCommit) {
}

func (replica *replica) handleCommit(commit Commit) {
	replica.blockchain.Extend(commit.Commit)
}

func (replica *replica) shouldDropTransition(transition Transition) bool {
	switch transition := transition.(type) {
	case Proposed:
		if transition.Height < replica.state.Height() {
			return true
		}
	case PreVoted:
		if transition.Height < replica.state.Height() {
			return true
		}
	case PreCommitted:
		if transition.Polka.Height < replica.state.Height() {
			return true
		}
	}
	return false
}

func (replica *replica) shouldBufferTransition(transition Transition) bool {
	switch transition := transition.(type) {
	case Proposed:
		if transition.Height > replica.state.Height() {
			return true
		}
	case PreVoted:
		if transition.Height > replica.state.Height() {
			return true
		}
	case PreCommitted:
		if transition.Polka.Height > replica.state.Height() {
			return true
		}
	}
	return false
}
