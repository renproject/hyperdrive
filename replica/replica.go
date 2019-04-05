package replica

import (
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/state"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
)

type Dispatcher interface {
	Dispatch(shardHash sig.Hash, action state.Action)
}

type Replica interface {
	Init()
	State() state.State
	Transact(transaction tx.Transaction)
	Transition(transition state.Transition)
}

type replica struct {
	dispatcher Dispatcher

	signer           sig.Signer
	validator        Validator
	txPool           tx.Pool
	state            state.State
	stateMachine     state.Machine
	transitionBuffer state.TransitionBuffer
	blockchain       *block.Blockchain
	shard            shard.Shard
}

func New(dispatcher Dispatcher, signer sig.SignerVerifier, txPool tx.Pool, state state.State, stateMachine state.Machine, transitionBuffer state.TransitionBuffer, blockchain *block.Blockchain, shard shard.Shard) Replica {
	replica := &replica{
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

func (replica *replica) State() state.State {
	return replica.state
}

func (replica *replica) Transact(tx tx.Transaction) {
	replica.txPool.Enqueue(tx)
}

func (replica *replica) Transition(transition state.Transition) {
	if replica.shouldDropTransition(transition) {
		return
	}
	if replica.shouldBufferTransition(transition) {
		replica.transitionBuffer.Enqueue(transition)
		return
	}
	for ok := true; ok; transition, ok = replica.transitionBuffer.Dequeue(replica.state.Height()) {
		if !replica.isTransitionValid(transition) {
			continue
		}
		action := replica.transition(transition)
		if action != nil {
			// It is important that the Action is dispatched after the State has been completely transitioned in the
			// Replica. Otherwise, re-entrance into the Replica may cause issues.
			replica.dispatchAction(action)
		}
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
		replica.dispatcher.Dispatch(replica.shard.Hash, state.SignedPreCommit{
			SignedPreCommit: signedPreCommit,
		})
	case state.Commit:
		replica.dispatcher.Dispatch(replica.shard.Hash, action)
		replica.blockchain.Extend(action.Commit)
		replica.generateSignedBlock()
	}
}

func (replica *replica) isTransitionValid(transition state.Transition) bool {
	switch transition := transition.(type) {
	case state.Proposed:
		return replica.validator.ValidateBlock(transition.SignedBlock)
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
		if transition.Height < replica.state.Height() {
			return true
		}
	case state.PreVoted:
		if transition.Height < replica.state.Height() {
			return true
		}
	case state.PreCommitted:
		if transition.Polka.Height < replica.state.Height() {
			return true
		}
	}
	return false
}

func (replica *replica) shouldBufferTransition(transition state.Transition) bool {
	switch transition := transition.(type) {
	case state.Proposed:
		if transition.Height <= replica.state.Height() {
			return false
		}
	case state.PreVoted:
		if transition.Height <= replica.state.Height() {
			return false
		}
	case state.PreCommitted:
		if transition.Polka.Height <= replica.state.Height() {
			return false
		}
	case state.TimedOut:
		// TimedOut transitions are never buffered
		return false
	}
	return true
}

func (replica *replica) shouldProposeBlock() bool {
	return replica.signer.Signatory().Equal(replica.shard.Leader(replica.state.Round()))
}

func (replica *replica) generateSignedBlock() {
	if replica.shouldProposeBlock() {
		propose := state.Propose{
			SignedBlock: replica.buildSignedBlock(),
		}
		replica.dispatcher.Dispatch(replica.shard.Hash, propose)

		// It is important that the Action is dispatched after the State has been completely transitioned in the
		// Replica. Otherwise, re-entrance into the Replica may cause issues.
		replica.dispatchAction(replica.transition(state.Proposed{
			SignedBlock: propose.SignedBlock,
		}))
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
		// Check invariant
		panic("invariant violated: blockchain has no head")
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

func (replica *replica) transition(transition state.Transition) state.Action {
	nextState, action := replica.stateMachine.Transition(replica.state, transition)
	replica.state = nextState
	replica.transitionBuffer.Drop(replica.state.Height())
	return action
}
