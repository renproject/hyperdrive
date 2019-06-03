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
			replica.stateMachine.SyncCommit(commit)
		}
		return true
	}
	return replica.previousShardValidator.ValidateCommit(commit)
}

func (replica *replica) Transition(transition state.Transition) {
	if replica.shouldDropTransition(transition) {
		fmt.Printf("(Height =%d) %T dropping %T (Round=%d)\n", replica.stateMachine.Height(), replica.stateMachine.State(), transition, transition.Round())
		return
	}
	if replica.shouldBufferTransition(transition) {
		fmt.Printf("(Height =%d) %T buffering %T (Round=%d)\n", replica.stateMachine.Height(), replica.stateMachine.State(), transition, transition.Round())
		replica.transitionBuffer.Enqueue(transition)
		return
	}
	for ok := true; ok; transition, ok = replica.transitionBuffer.Dequeue(replica.stateMachine.Height()) {
		if !replica.isTransitionValid(transition) {
			fmt.Printf("(Height =%d) %T invalid %T (Round=%d)\n", replica.stateMachine.Height(), replica.stateMachine.State(), transition, transition.Round())
			continue
		}

		fmt.Printf("(Height =%d) %T received valid %T (Round=%d)\n", replica.stateMachine.Height(), replica.stateMachine.State(), transition, transition.Round())

		replica.dispatchAction(replica.transition(transition))
	}
}

func (replica *replica) dispatchAction(action state.Action) {
	fmt.Printf("(Height =%d) dispatching %T \n", replica.stateMachine.Height(), action)
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
		if len(action.Commit.Signatures) > 0 {
			replica.dispatcher.Dispatch(replica.shardHash, action.Commit)
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
		return replica.validator.ValidatePropose(transition.SignedPropose, nil)
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
		fmt.Printf("%T with H=%d Received proposal for (H=%d, R=%d)\n", replica.stateMachine.State(), replica.stateMachine.Height(), transition.Block.Height, transition.Round())
		// Only buffer Proposals from the future
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
