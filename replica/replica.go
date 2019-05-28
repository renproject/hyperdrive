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

// NumTicksToTriggerTimeOut specifies the maximum number of Ticks to wait before
// triggering a TimedOut  transition.
const NumTicksToTriggerTimeOut = 2

type Dispatcher interface {
	Dispatch(shardHash sig.Hash, action state.Action)
}

type Replica interface {
	Init()
	Transition(transition state.Transition)
	SyncCommit(commit block.Commit) bool
}

type replica struct {
	index      int
	dispatcher Dispatcher

	ticks                  int
	signer                 sig.Signer
	validator              Validator
	previousShardValidator Validator
	txPool                 tx.Pool
	stateMachine           state.Machine
	transitionBuffer       state.TransitionBuffer
	shard                  shard.Shard
	lastBlock              *block.SignedBlock
}

func New(index int, dispatcher Dispatcher, signer sig.SignerVerifier, txPool tx.Pool, stateMachine state.Machine, transitionBuffer state.TransitionBuffer, shard, previousShard shard.Shard, lastBlock block.SignedBlock) Replica {
	replica := &replica{
		index:      index,
		dispatcher: dispatcher,

		ticks:                  0,
		signer:                 signer,
		validator:              NewValidator(signer, shard),
		previousShardValidator: NewValidator(signer, previousShard),
		txPool:                 txPool,
		stateMachine:           stateMachine,
		transitionBuffer:       transitionBuffer,
		shard:                  shard,
		lastBlock:              &lastBlock,
	}
	return replica
}

func (replica *replica) Init() {
	propose := replica.stateMachine.StartRound(0, nil)
	// fmt.Println(replica.index, "yo yo got propose", propose)
	if propose, ok := propose.(state.Propose); ok {
		// fmt.Println(replica.index, "got propose", propose)
		replica.dispatcher.Dispatch(replica.shard.Hash, state.Propose{
			SignedPropose: propose.SignedPropose,
		})
		replica.Transition(state.Proposed{
			SignedPropose: propose.SignedPropose,
		})

	}
}

func (replica *replica) SyncCommit(commit block.Commit) bool {
	if replica.validator.ValidateCommit(commit) {
		if replica.lastBlock.Height < commit.Polka.Height {
			replica.stateMachine.SyncCommit(commit)
			replica.lastBlock = commit.Polka.Block
		}
		return true
	}
	return replica.previousShardValidator.ValidateCommit(commit)
}

func (replica *replica) Transition(transition state.Transition) {
	if replica.shouldDropTransition(transition) {
		// fmt.Printf("dropping transition %T\n", transition)
		return
	}
	if replica.shouldBufferTransition(transition) {
		// fmt.Printf("buffering transition %T\n", transition)
		replica.transitionBuffer.Enqueue(transition)
		return
	}

	for ok := true; ok; transition, ok = replica.transitionBuffer.Dequeue(replica.stateMachine.Height()) {
		// if !replica.isTransitionValid(transition) {
		// 	continue
		// }

		fmt.Printf("%d got transition %T\n", replica.index, transition)
		action := replica.transition(transition)
		fmt.Printf("%d got action %T\n", replica.index, action)
		// It is important that the Action is dispatched after the State has been completely transitioned in the
		// Replica. Otherwise, re-entrance into the Replica may cause issues.
		if propose, ok := action.(state.Propose); ok {
			// It is important that the Action is dispatched after the State has been completely transitioned in the
			// Replica. Otherwise, re-entrance into the Replica may cause issues.
			replica.dispatcher.Dispatch(replica.shard.Hash, state.Propose{
				SignedPropose: propose.SignedPropose,
			})
			replica.Transition(state.Proposed{
				SignedPropose: propose.SignedPropose,
			})
			continue

		}
		replica.dispatchAction(action)
	}
}

func (replica *replica) dispatchAction(action state.Action) {
	if action == nil {
		return
	}

	switch action := action.(type) {
	case state.SignedPreVote:
		replica.dispatcher.Dispatch(replica.shard.Hash, state.SignedPreVote{
			SignedPreVote: action.SignedPreVote,
		})
	case state.SignedPreCommit:
		replica.dispatcher.Dispatch(replica.shard.Hash, state.SignedPreCommit{
			SignedPreCommit: action.SignedPreCommit,
		})
	case state.Commit:
		if action.Commit.Polka.Block != nil {
			replica.lastBlock = action.Commit.Polka.Block
			replica.dispatcher.Dispatch(replica.shard.Hash, action)
		}
	default:
		panic(fmt.Sprintf("unexpected message %T", action))
	}
}

func (replica *replica) isTransitionValid(transition state.Transition) bool {
	switch transition := transition.(type) {
	case state.Proposed:
		return replica.validator.ValidatePropose(transition.SignedPropose, nil)
	case state.PreVoted:
		return replica.validator.ValidatePreVote(transition.SignedPreVote, nil)
	case state.PreCommitted:
		return replica.validator.ValidatePreCommit(transition.SignedPreCommit, nil)
	case state.Ticked:
		return transition.Time.Before(time.Now())
	}
	return false
}

func (replica *replica) shouldDropTransition(transition state.Transition) bool {
	switch transition := transition.(type) {
	case state.Proposed:
		if transition.Block.Height < replica.stateMachine.Height() || transition.Round() < replica.stateMachine.Round() {
			return true
		}
	case state.PreVoted:
		if transition.Height < replica.stateMachine.Height() || transition.Round() < replica.stateMachine.Round() {
			return true
		}
	case state.PreCommitted:
		if transition.Polka.Height < replica.stateMachine.Height() || transition.Round() < replica.stateMachine.Round() {
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
		return true
	default:
		return false
	}
}

func (replica *replica) transition(transition state.Transition) state.Action {
	action := replica.stateMachine.Transition(transition)
	replica.transitionBuffer.Drop(replica.stateMachine.Height())
	if commit, ok := action.(state.Commit); ok && commit.Polka.Block != nil {
		// If round has progressed, drop all prevotes and precommits in the state-machine
		replica.stateMachine.Drop()
	}
	if propose, ok := action.(state.Propose); ok && len(propose.Commit.Signatures) > 0 && propose.Commit.Polka.Block != nil {
		// If round has progressed, drop all prevotes and precommits in the state-machine
		replica.stateMachine.Drop()
	}
	return action
}
