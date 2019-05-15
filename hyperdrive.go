package hyperdrive

import (
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/state"
	"github.com/renproject/hyperdrive/tx"
)

// NumHistoricalShards specifies the number of historical shards allowed.
const NumHistoricalShards = 3

// NumTicksToTriggerTimeOut specifies the maximum number of Ticks to wait before
// triggering a TimedOut  transition.
const NumTicksToTriggerTimeOut = 2

// Hyperdrive accepts blocks and ticks and sends relevant Transitions to the respective replica.
type Hyperdrive interface {
	AcceptTick(t time.Time)
	AcceptPropose(shardHash sig.Hash, proposed block.SignedBlock)
	AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote)
	AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit)
	AcceptShard(shard shard.Shard, head block.SignedBlock, pool tx.Pool)
}

type hyperdrive struct {
	signer     sig.SignerVerifier
	dispatcher replica.Dispatcher

	shardReplicas map[sig.Hash]replica.Replica
	shardHistory  []sig.Hash

	ticksPerShard map[sig.Hash]int
}

// New returns a Hyperdrive.
func New(signer sig.SignerVerifier, dispatcher replica.Dispatcher) Hyperdrive {
	return &hyperdrive{
		signer:     signer,
		dispatcher: dispatcher,

		shardReplicas: map[sig.Hash]replica.Replica{},
		shardHistory:  []sig.Hash{},

		ticksPerShard: map[sig.Hash]int{},
	}
}

func (hyperdrive *hyperdrive) AcceptTick(t time.Time) {
	// 1. Increment number of ticks seen by each shard
	for shardHash := range hyperdrive.shardReplicas {
		hyperdrive.ticksPerShard[shardHash]++

		if hyperdrive.ticksPerShard[shardHash] > NumTicksToTriggerTimeOut {
			// 2. Send a TimedOut transition to the shard
			if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
				replica.Transition(state.TimedOut{Time: t})
				hyperdrive.ticksPerShard[shardHash] = 0 // Reset tickPerShard
			}
		}
	}
}

func (hyperdrive *hyperdrive) AcceptPropose(shardHash sig.Hash, proposed block.SignedBlock) {
	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		hyperdrive.ticksPerShard[shardHash] = 0 // Reset tickPerShard
		replica.Transition(state.Proposed{SignedBlock: proposed})
	}
}

func (hyperdrive *hyperdrive) AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote) {
	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		replica.Transition(state.PreVoted{SignedPreVote: preVote})
	}
}

func (hyperdrive *hyperdrive) AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit) {
	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		replica.Transition(state.PreCommitted{SignedPreCommit: preCommit})
	}
}

func (hyperdrive *hyperdrive) AcceptShard(shard shard.Shard, head block.SignedBlock, pool tx.Pool) {
	if _, ok := hyperdrive.shardReplicas[shard.Hash]; ok {
		return
	}

	r := replica.New(
		hyperdrive.dispatcher,
		hyperdrive.signer,
		pool,
		state.NewMachine(state.WaitingForPropose{}, block.NewPolkaBuilder(), block.NewCommitBuilder(), shard.ConsensusThreshold()),
		shard,
		head,
	)

	hyperdrive.shardReplicas[shard.Hash] = r
	hyperdrive.shardHistory = append(hyperdrive.shardHistory, shard.Hash)
	hyperdrive.ticksPerShard[shard.Hash] = 0
	if len(hyperdrive.shardHistory) > NumHistoricalShards {
		delete(hyperdrive.shardReplicas, hyperdrive.shardHistory[0])
		hyperdrive.shardHistory = hyperdrive.shardHistory[1:]
	}

	r.Init()
}
