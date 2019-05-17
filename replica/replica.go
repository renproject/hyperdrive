package replica

import (
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/state"
	"github.com/renproject/hyperdrive/tx"
)

type Dispatcher interface {
	Dispatch(shardHash sig.Hash, action state.Action)
}

type Replica interface {
	Init()
	Transition(transition state.Transition)
	SyncCommits(commits []block.Commit)
}

type replica struct {
	dispatcher Dispatcher

	signer           sig.Signer
	validator        Validator
	txPool           tx.Pool
	stateMachine     state.Machine
	transitionBuffer state.TransitionBuffer
	shard            shard.Shard
	lastBlock        *block.SignedBlock
}

func New(dispatcher Dispatcher, signer sig.SignerVerifier, txPool tx.Pool, stateMachine state.Machine, transitionBuffer state.TransitionBuffer, shard shard.Shard, lastBlock block.SignedBlock) Replica {
	replica := &replica{
		dispatcher: dispatcher,

		signer:           signer,
		validator:        NewValidator(signer, shard),
		txPool:           txPool,
		stateMachine:     stateMachine,
		transitionBuffer: transitionBuffer,
		shard:            shard,
		lastBlock:        &lastBlock,
	}
	return replica
}

func (replica *replica) Init() {
	replica.generateSignedBlock()
}

func (replica *replica) SyncCommits(commits []block.Commit) {
	fmt.Printf("current last block height %d\n", replica.lastBlock.Height)
	for _, commit := range commits {
		// TODO: enable validation for commits; Figure out a way to store the commits in the blockchain.
		// if replica.validator.ValidateCommit(commit) {
		if replica.lastBlock.Height < commit.Polka.Height {
			replica.lastBlock = commit.Polka.Block
		}
		// }
	}
	fmt.Printf("new last block height %d\n", replica.lastBlock.Height)
}

func (replica *replica) Transition(transition state.Transition) {
	if replica.shouldDropTransition(transition) {
		return
	}
	if replica.shouldBufferTransition(transition) {
		replica.transitionBuffer.Enqueue(transition)
		return
	}
	for ok := true; ok; transition, ok = replica.transitionBuffer.Dequeue(replica.stateMachine.Height()) {
		if !replica.isTransitionValid(transition) {
			continue
		}
		action := replica.transition(transition)
		// It is important that the Action is dispatched after the State has been completely transitioned in the
		// Replica. Otherwise, re-entrance into the Replica may cause issues.
		replica.dispatchAction(action)
	}
}

func (replica *replica) dispatchAction(action state.Action) {
	if action == nil {
		return
	}

	switch action := action.(type) {
	case state.PreVote:
		signedPreVote, err := action.PreVote.Sign(replica.signer)
		if err != nil {
			// FIXME: We should handle this error properly. It would not make sense to propagate it, but there should at
			// least be some sane logging and recovery.
			panic(err)
		}
		replica.stateMachine.InsertPrevote(signedPreVote)
		replica.dispatcher.Dispatch(replica.shard.Hash, state.SignedPreVote{
			SignedPreVote: signedPreVote,
		})
	case state.PreCommit:
		signedPreCommit, err := action.PreCommit.Sign(replica.signer)
		if err != nil {
			// FIXME: We should handle this error properly. It would not make sense to propagate it, but there should at
			// least be some sane logging and recovery.
			panic(err)
		}
		replica.stateMachine.InsertPrecommit(signedPreCommit)
		replica.dispatcher.Dispatch(replica.shard.Hash, state.SignedPreCommit{
			SignedPreCommit: signedPreCommit,
		})
	case state.Commit:
		if action.Commit.Polka.Block != nil {
			replica.lastBlock = action.Commit.Polka.Block
			replica.dispatcher.Dispatch(replica.shard.Hash, action)
		}
		replica.generateSignedBlock()
	}
}

func (replica *replica) isTransitionValid(transition state.Transition) bool {
	switch transition := transition.(type) {
	case state.Proposed:
		return replica.validator.ValidatePropose(transition.SignedPropose, replica.lastBlock)
	case state.PreVoted:
		return replica.validator.ValidatePreVote(transition.SignedPreVote)
	case state.PreCommitted:
		return replica.validator.ValidatePreCommit(transition.SignedPreCommit)
	case state.TimedOut:
		return transition.Time.Before(time.Now())
	}
	return false
}

func (replica *replica) shouldDropTransition(transition state.Transition) bool {
	switch transition := transition.(type) {
	case state.Proposed:
		if transition.Block.Height < replica.stateMachine.Height() {
			return true
		}
	case state.PreVoted:
		if transition.Height < replica.stateMachine.Height() {
			return true
		}
	case state.PreCommitted:
		if transition.Polka.Height < replica.stateMachine.Height() {
			return true
		}
	}
	return false
}

func (replica *replica) shouldBufferTransition(transition state.Transition) bool {
	switch transition := transition.(type) {
	case state.Proposed:
		// Only buffer Proposals from the future
		if transition.Block.Height <= replica.stateMachine.Height() {
			return false
		}
		fmt.Printf("buffering propose: %d, current height: %d\n", transition.Block.Height, replica.stateMachine.Height())
		return true
	default:
		return false
	}
}

func (replica *replica) shouldProposeBlock() bool {
	return replica.signer.Signatory().Equal(replica.shard.Leader(replica.stateMachine.Round()))
}

func (replica *replica) generateSignedBlock() {
	if replica.shouldProposeBlock() {
		propose := block.Propose{
			Block: replica.buildSignedBlock(),
			Round: replica.stateMachine.Round(),
		}

		signedPropose, err := propose.Sign(replica.signer)
		if err != nil {
			panic(err)
		}
		replica.dispatcher.Dispatch(replica.shard.Hash, state.Propose{
			SignedPropose: signedPropose,
		})

		// It is important that the Action is dispatched after the State has been completely transitioned in the
		// Replica. Otherwise, re-entrance into the Replica may cause issues.
		replica.dispatchAction(replica.transition(state.Proposed{
			SignedPropose: signedPropose,
		}))
	}
}

func (replica *replica) buildSignedBlock() block.SignedBlock {
	// TODO: We should put more than one transaction into a block.
	transactions := make(tx.Transactions, 0, block.MaxTransactions)
	transaction, ok := replica.txPool.Dequeue()
	for ok && len(transactions) < block.MaxTransactions {
		transactions = append(transactions, transaction)
		transaction, ok = replica.txPool.Dequeue()
	}

	block := block.New(
		replica.stateMachine.Height(),
		replica.lastBlock.Header,
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

func (replica *replica) transition(transition state.Transition) state.Action {
	action := replica.stateMachine.Transition(transition)
	replica.transitionBuffer.Drop(replica.stateMachine.Height())
	if commit, ok := action.(state.Commit); ok && commit.Polka.Block != nil {
		// If round has progressed, drop all prevotes and precommits in the state-machine
		replica.stateMachine.Drop()
	}
	return action
}
