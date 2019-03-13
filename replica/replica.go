// Package replica core package
//
// I assume that all provided `Transition`s are well formed and valid by
// the time they reach this module.
package replica

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
)

type Dispatcher interface {
	Dispatch(action consensus.Action)
}

type Replica interface {
	Shard(shard shard.Shard)
	Transact(transaction tx.Transaction)
	Transition(transition consensus.Transition)
}

type replica struct {
	dispatcher Dispatcher

	signer           sig.Signer
	txPool           tx.Pool
	state            consensus.State
	stateMachine     consensus.StateMachine
	transitionBuffer consensus.TransitionBuffer
	blockchain       block.Blockchain
}

func New(
	dispatcher Dispatcher,
	signer sig.Signer,
	txPool tx.Pool,
	state consensus.State,
	stateMachine consensus.StateMachine,
	transitionBuffer consensus.TransitionBuffer,
	blockchain block.Blockchain,
) Replica {
	return &replica{
		dispatcher:       dispatcher,
		state:            state,
		stateMachine:     stateMachine,
		transitionBuffer: transitionBuffer,
		blockchain:       blockchain,
	}
}

func (replica *replica) Shard(shard shard.Shard) {
	// TODO: There is still a lot to figure out here.
}

func (replica *replica) Transact(tx tx.Transaction) {
	replica.txPool.Enqueue(tx)
}

func (replica *replica) Transition(transition consensus.Transition) {
	if replica.shouldDropTransition(transition) {
		return
	}
	if replica.shouldBufferTransition(transition) {
		replica.transitionBuffer.Enqueue(transition)
		return
	}
	for ok := true; ok; transition, ok = replica.transitionBuffer.Dequeue(replica.state.Height()) {
		nextState, action := replica.stateMachine.Transition(replica.state, transition)
		replica.state = nextState
		replica.transitionBuffer.Drop(replica.state.Height())
		// WARNING: It is important that the Action is dispatched after the State has been completely transitioned in
		// the Replica. Otherwise, re-entrance into the Replica may cause issues.
		replica.dispatchAction(action)
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
	// Passthrough
}

func (replica *replica) handlePreCommit(preCommit consensus.PreCommit) {
	// Passthrough
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
