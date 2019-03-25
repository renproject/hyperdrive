package replica

import (
	"log"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/tx"
)

type Dispatcher interface {
	Dispatch(action consensus.Action)
}

type Replica interface {
	Transact(transaction tx.Transaction)
	Transition(transition consensus.Transition)
}

type replica struct {
	dispatcher Dispatcher

	signatory        sig.Signatory
	txPool           tx.Pool
	state            consensus.State
	stateMachine     consensus.StateMachine
	transitionBuffer consensus.TransitionBuffer
	blockchain       block.Blockchain
	shard            shard.Shard
}

func New(
	dispatcher Dispatcher,
	signatory sig.Signatory,
	txPool tx.Pool,
	state consensus.State,
	stateMachine consensus.StateMachine,
	transitionBuffer consensus.TransitionBuffer,
	blockchain block.Blockchain,
	shard shard.Shard,
) Replica {
	replica := &replica{
		dispatcher: dispatcher,

		signatory:        signatory,
		txPool:           txPool,
		state:            state,
		stateMachine:     stateMachine,
		transitionBuffer: transitionBuffer,
		blockchain:       blockchain,
		shard:            shard,
	}
	return replica
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
		// It is important that the Action is dispatched after the State has been completely transitioned in the
		// Replica. Otherwise, re-entrance into the Replica may cause issues.
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
	if replica.shouldProposeBlock() {
		replica.dispatcher.Dispatch(consensus.Propose{
			Block: replica.generateBlock(),
		})
	}
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
		if transition.Height <= replica.state.Height() {
			return false
		}
	case consensus.PreVoted:
		if transition.Height <= replica.state.Height() {
			return false
		}
	case consensus.PreCommitted:
		if transition.Polka.Height <= replica.state.Height() {
			return false
		}
	}
	return true
}

func (replica *replica) shouldProposeBlock() bool {
	return replica.signatory.Equal(replica.shard.Leader(replica.state.Round()))
}

func (replica *replica) generateBlock() block.Block {
	// TODO: Generate a Block using the transaction Pool, current Blockchain, and current Shard.
	transaction, ok := replica.txPool.Dequeue()
	if !ok {
		return block.Block{}
	}

	parent, ok := replica.blockchain.Head()
	if !ok {
		parent = block.Genesis()
	}

	newBlock := block.Block{
		Time:         time.Now(),
		Round:        replica.blockchain.Round() + 1,
		Height:       replica.blockchain.Height() + 1,
		Header:       replica.shard.Hash,
		ParentHeader: parent.Header,
		Signatory:    replica.signer.Signatory(),
		Txs:          []tx.Transaction{transaction}, // TODO: get all pending txs in the pool
	}

	var err error
	// TODO: (review) Sign the entire block?
	newBlock.Signature, err = replica.signer.Sign(ecdsa.Hash([]byte(newBlock.String())))
	if err != nil {
		log.Println(err)
		return block.Block{}
	}
	return newBlock

}
