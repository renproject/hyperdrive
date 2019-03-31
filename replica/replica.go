package replica

import (
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
)

type Dispatcher interface {
	Dispatch(shardHash sig.Hash, action consensus.Action)
}

type Replica interface {
	Init()
	State() consensus.State
	Transact(transaction tx.Transaction)
	Transition(transition consensus.Transition)
}

type replica struct {
	i          int
	dispatcher Dispatcher

	signer           sig.Signer
	validator        Validator
	txPool           tx.Pool
	state            consensus.State
	stateMachine     consensus.StateMachine
	transitionBuffer consensus.TransitionBuffer
	blockchain       *block.Blockchain
	shard            shard.Shard
}

func New(index int,
	dispatcher Dispatcher,
	signer sig.SignerVerifier,
	txPool tx.Pool,
	state consensus.State,
	stateMachine consensus.StateMachine,
	transitionBuffer consensus.TransitionBuffer,
	blockchain *block.Blockchain,
	shard shard.Shard,
) Replica {
	replica := &replica{
		i:          index,
		dispatcher: dispatcher,

		signer:           signer,
		validator:        NewValidator(signer, shard, blockchain),
		txPool:           txPool,
		state:            state,
		stateMachine:     stateMachine,
		transitionBuffer: transitionBuffer,
		blockchain:       blockchain,
		shard:            shard,
	}
	return replica
}

func (replica *replica) Init() {
	replica.generateSignedBlock()
}

func (replica *replica) State() consensus.State {
	return replica.state
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
		// TODO: is this where transitions should be validated?
		if !replica.isTransitionValid(transition) {
			return
		}
		nextState, action := replica.stateMachine.Transition(replica.state, transition)
		if action != nil {
			replica.state = nextState
			replica.transitionBuffer.Drop(replica.state.Height())
			// It is important that the Action is dispatched after the State has been completely transitioned in the
			// Replica. Otherwise, re-entrance into the Replica may cause issues.
			replica.dispatchAction(action)
		}
	}
}

func (replica *replica) dispatchAction(action consensus.Action) {
	if action == nil {
		return
	}
	var dispatchAction consensus.Action
	switch action := action.(type) {
	case consensus.PreVote:
		signedPreVote, err := action.PreVote.Sign(replica.signer)
		if err != nil {
			// FIXME: We should handle this error properly. It would not make sense to propagate it, but there should at
			// least be some sane logging and recovery.
			panic(err)
		}
		dispatchAction = consensus.SignedPreVote{
			SignedPreVote: signedPreVote,
		}
		replica.handlePreVote(consensus.SignedPreVote{
			SignedPreVote: signedPreVote,
		})
	case consensus.PreCommit:
		signedPreCommit, err := action.PreCommit.Sign(replica.signer)
		if err != nil {
			// FIXME: We should handle this error properly. It would not make sense to propagate it, but there should at
			// least be some sane logging and recovery.
			panic(err)
		}
		dispatchAction = consensus.SignedPreCommit{
			SignedPreCommit: signedPreCommit,
		}
		replica.handlePreCommit(consensus.SignedPreCommit{
			SignedPreCommit: signedPreCommit,
		})
	case consensus.Commit:
		if replica.i == 0 {
			fmt.Printf("got commit: %x Height: %d\n", action.Polka.Block.Header, action.Polka.Block.Height)
		}
		dispatchAction = action
		replica.handleCommit(action)
	}
	replica.dispatcher.Dispatch(replica.shard.Hash, dispatchAction)
}

func (replica *replica) handlePreVote(preVote consensus.SignedPreVote) {
	// Passthrough
}

func (replica *replica) handlePreCommit(preCommit consensus.SignedPreCommit) {
	// Passthrough
}

func (replica *replica) handleCommit(commit consensus.Commit) {
	replica.blockchain.Extend(commit.Commit)
	replica.generateSignedBlock()
}

func (replica *replica) isTransitionValid(transition consensus.Transition) bool {
	switch transition := transition.(type) {
	case consensus.Proposed:
		return replica.validator.ValidateBlock(transition.SignedBlock)
	case consensus.PreVoted:
		return replica.validator.ValidatePreVote(transition.SignedPreVote)
	case consensus.PreCommitted:
		return replica.validator.ValidatePreCommit(transition.SignedPreCommit)
	case consensus.TimedOut:
		return !transition.Time.After(time.Now())
	}
	return false
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
	return replica.signer.Signatory().Equal(replica.shard.Leader(replica.state.Round()))
}

func (replica *replica) generateSignedBlock() {
	if replica.shouldProposeBlock() {
		replica.dispatcher.Dispatch(replica.shard.Hash, consensus.Propose{
			SignedBlock: replica.buildSignedBlock(),
		})
	}
}

func (replica *replica) buildSignedBlock() block.SignedBlock {
	// TODO: We should put more than one transaction into a block.
	transactions := tx.Transactions{}
	transaction, ok := replica.txPool.Dequeue()
	if ok {
		transactions = append(transactions, transaction)
	}

	parent, ok := replica.blockchain.Head()
	if !ok {
		parent = block.Genesis()
	}

	block := block.New(
		replica.state.Round(),
		replica.state.Height(),
		parent.Header,
		transactions,
	)
	signedBlock, err := block.Sign(replica.signer)
	if err != nil {
		// FIXME: We should handle this error properly. It would not make sense to propagate it, but there should at
		// least be some sane logging and recovery.
		panic(err)
	}
	return signedBlock
}
