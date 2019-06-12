package replica

import (
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/state"
)

type Dispatcher interface {
	Dispatch(shardHash sig.Hash, action state.Action)
}

type Replica interface {
	Init()
	Transition(transition state.Transition)
	SyncCommit(commit block.Commit) bool
}

type replica struct {
	dispatcher Dispatcher

	validator              Validator
	previousShardValidator Validator
	stateMachine           state.Machine
	transitionBuffer       state.TransitionBuffer
	shardHash              sig.Hash
}

func New(dispatcher Dispatcher, signer sig.SignerVerifier, stateMachine state.Machine, transitionBuffer state.TransitionBuffer, shard, previousShard shard.Shard) Replica {
	replica := &replica{
		dispatcher: dispatcher,

		validator:              NewValidator(signer, shard),
		previousShardValidator: NewValidator(signer, previousShard),
		stateMachine:           stateMachine,
		transitionBuffer:       transitionBuffer,
		shardHash:              shard.Hash,
	}
	return replica
}

func (replica *replica) Init() {
	replica.dispatchAction(replica.stateMachine.StartRound(0, nil))
}

func (replica *replica) SyncCommit(commit block.Commit) bool {
	if replica.validator.ValidateCommit(commit) {
		if replica.stateMachine.LastBlock().Height < commit.Polka.Height {
			replica.stateMachine.Commit(commit)
		}
		return true
	}
	return replica.previousShardValidator.ValidateCommit(commit)
}

func (replica *replica) Transition(transition state.Transition) {
	for ok := true; ok; transition, ok = replica.transitionBuffer.Dequeue(replica.stateMachine.Height()) {
		if replica.shouldDropTransition(transition) {
			continue
		}

		if !replica.isTransitionValid(transition) {
			continue
		}

		// If a `Proposed` is seen for a higher round, buffer and return immediately
		if replica.shouldBufferTransition(transition) {
			replica.transitionBuffer.Enqueue(transition)
			return
		}

		replica.dispatchAction(replica.transition(transition))
	}
}

func (replica *replica) dispatchAction(action state.Action) {
	if action == nil {
		return
	}

	switch action := action.(type) {
	case state.Propose:
		// Dispatch the Propose
		replica.dispatcher.Dispatch(replica.shardHash, state.Propose{
			SignedPropose: action.SignedPropose,
		})
		// Dispatch commits (if any)
		if len(action.LastCommit.Signatures) > 0 {
			replica.dispatcher.Dispatch(replica.shardHash, action.LastCommit)
		}
		// Transition the stateMachine
		replica.Transition(state.Proposed{
			SignedPropose: action.SignedPropose,
		})

	case state.SignedPreVote:
		replica.dispatcher.Dispatch(replica.shardHash, state.SignedPreVote{
			SignedPreVote: action.SignedPreVote,
		})
	case state.SignedPreCommit:
		replica.dispatcher.Dispatch(replica.shardHash, state.SignedPreCommit{
			SignedPreCommit: action.SignedPreCommit,
		})
	case state.Commit:
		if action.Commit.Polka.Block != nil {
			replica.dispatcher.Dispatch(replica.shardHash, action)
		}
	default:
		panic(fmt.Sprintf("unexpected message %T", action))
	}
}

func (replica *replica) isTransitionValid(transition state.Transition) bool {
	switch transition := transition.(type) {
	case state.Proposed:
		return replica.validator.ValidatePropose(transition.SignedPropose)
	case state.PreVoted:
		return replica.validator.ValidatePreVote(transition.SignedPreVote)
	case state.PreCommitted:
		return replica.validator.ValidatePreCommit(transition.SignedPreCommit)
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
		// Only buffer Proposals for higher rounds of the current height
		if transition.Block.Height == replica.stateMachine.Height() && transition.Round() > replica.stateMachine.Round() {
			return true
		}
		return false
	default:
		return false
	}
}

func (replica *replica) transition(transition state.Transition) state.Action {
	action := replica.stateMachine.Transition(transition)
	replica.transitionBuffer.Drop(replica.stateMachine.Height())
	return action
}
