package hyperdrive

import (
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
)

// NumHistoricalShards specifies the number of historical shards allowed.
const NumHistoricalShards = 3

// NumTicksToTriggerTimeOut specifies the maximum number of Ticks to wait before
// triggering a TimedOut  transition.
const NumTicksToTriggerTimeOut = 2

// Dispatcher is responsible for verifying and forwarding `Action`s to shards.
type Dispatcher struct {
	shard shard.Shard
}

// NewDispatcher returns a Dispatcher for the given `shard`.
func NewDispatcher(shard shard.Shard) replica.Dispatcher {
	return &Dispatcher{
		shard: shard,
	}
}

// Dispatch `action` to the shard.
func (d *Dispatcher) Dispatch(action consensus.Action) {
	// TODO:
	// 1. Broadcast the action to the entire shard
}

// Hyperdrive accepts blocks and ticks and sends relevant Transitions to the respective replica.
type Hyperdrive interface {
	AcceptTick(t time.Time)
	AcceptPropose(shardHash sig.Hash, proposed block.SignedBlock)
	AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote)
	AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit)
	AcceptShard(shard shard.Shard, blockchain block.Blockchain)
}

type hyperdrive struct {
	signer sig.SignerVerifier

	shardReplicas map[sig.Hash]replica.Replica
	shardHistory  []sig.Hash

	ticksPerShard map[sig.Hash]int
}

// New returns a Hyperdrive.
func New(signer sig.SignerVerifier) Hyperdrive {
	return &hyperdrive{
		signer: signer,

		shardReplicas: map[sig.Hash]replica.Replica{},
		shardHistory:  []sig.Hash{},

		ticksPerShard: map[sig.Hash]int{},
	}
}

func (hyperdrive *hyperdrive) AcceptTick(t time.Time) {
	// 1. Increment number of ticks seen by each shard
	for shardHash := range hyperdrive.shardReplicas {
		ticks := hyperdrive.ticksPerShard[shardHash]
		ticks++
		hyperdrive.ticksPerShard[shardHash] = ticks

		if ticks > NumTicksToTriggerTimeOut {
			// 2. Send a TimedOut transition to the shard
			if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
				replica.Transition(consensus.TimedOut{Time: t})
			}
		}
	}
}

func (hyperdrive *hyperdrive) AcceptPropose(shardHash sig.Hash, proposed block.SignedBlock) {
	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		replica.Transition(consensus.Proposed{SignedBlock: proposed})
	}
}

func (hyperdrive *hyperdrive) AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote) {
	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		replica.Transition(consensus.PreVoted{SignedPreVote: preVote})
	}
}

func (hyperdrive *hyperdrive) AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit) {
	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		replica.Transition(consensus.PreCommitted{SignedPreCommit: preCommit})
	}
}

func (hyperdrive *hyperdrive) AcceptShard(shard shard.Shard, blockchain block.Blockchain) {
	if _, ok := hyperdrive.shardReplicas[shard.Hash]; ok {
		return
	}

	r := replica.New(
		NewDispatcher(shard),
		hyperdrive.signer,
		tx.FIFOPool(),
		consensus.WaitForPropose(blockchain.Round(), blockchain.Height()),
		consensus.NewStateMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), shard.ConsensusThreshold()),
		consensus.NewTransitionBuffer(shard.Size()),
		blockchain,
		shard,
	)

	hyperdrive.shardReplicas[shard.Hash] = r
	hyperdrive.shardHistory = append(hyperdrive.shardHistory, shard.Hash)
	if len(hyperdrive.shardHistory) > NumHistoricalShards {
		delete(hyperdrive.shardReplicas, hyperdrive.shardHistory[0])
		hyperdrive.shardHistory = hyperdrive.shardHistory[1:]
	}

	r.Init()
}
