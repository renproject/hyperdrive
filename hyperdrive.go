package hyperdrive

import (
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
	"golang.org/x/crypto/sha3"
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

// Hyperdrive accepts, validates and pre-processes blocks and ticks and sends
// relevant Transitions to the respective replica.
type Hyperdrive interface {
	AcceptTick(t time.Time)
	AcceptPropose(shardHash sig.Hash, proposed block.SignedBlock)
	AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote)
	AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit)
	AcceptShard(shard shard.Shard, blockchain block.Blockchain)
}

type hyperdrive struct {
	signer sig.SignerVerifier

	shards        map[sig.Hash]shard.Shard
	shardReplicas map[sig.Hash]replica.Replica
	shardHistory  []sig.Hash

	ticksPerShard map[sig.Hash]int
}

// New returns a Hyperdrive.
func New(signer sig.SignerVerifier) Hyperdrive {
	return &hyperdrive{
		signer: signer,

		shards:        map[sig.Hash]shard.Shard{},
		shardReplicas: map[sig.Hash]replica.Replica{},
		shardHistory:  []sig.Hash{},

		ticksPerShard: map[sig.Hash]int{},
	}
}

func (hyperdrive *hyperdrive) AcceptTick(t time.Time) {
	// 1. Increment number of ticks seen by each shard
	for shardHash := range hyperdrive.shards {
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
	// 1. Verify the block is well-formed
	if !hyperdrive.validateBlock(shardHash, proposed) {
		return
	}

	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		replica.Transition(consensus.Proposed{SignedBlock: proposed})
	}
}

func (hyperdrive *hyperdrive) AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote) {
	// 1. Verify the pre-vote is well-formed
	if preVote.String() == (block.SignedPreVote{}).String() {
		return
	}

	if preVote.PreVote.Block != nil {
		if !hyperdrive.validateBlock(shardHash, *preVote.PreVote.Block) {
			return
		}
	}

	if preVote.PreVote.Round < 0 || preVote.PreVote.Height < 0 {
		return
	}

	// 2. Verify the signatory of the pre-vote
	if !hyperdrive.verifySignature(shardHash, []byte(preVote.PreVote.String()), preVote.Signature, preVote.Signatory) {
		return
	}

	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		replica.Transition(consensus.PreVoted{SignedPreVote: preVote})
	}
}

func (hyperdrive *hyperdrive) AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit) {
	// 1. Verify the pre-commit is well-formed
	if preCommit.String() == (block.SignedPreCommit{}).String() {
		return
	}

	if !hyperdrive.validatePolka(shardHash, preCommit.PreCommit.Polka) {
		return
	}

	// 2. Verify the signatory of the pre-commit
	if !hyperdrive.verifySignature(shardHash, []byte(preCommit.PreCommit.String()), preCommit.Signature, preCommit.Signatory) {
		return
	}

	if replica, ok := hyperdrive.shardReplicas[shardHash]; ok {
		replica.Transition(consensus.PreCommitted{SignedPreCommit: preCommit})
	}
}

func (hyperdrive *hyperdrive) AcceptShard(shard shard.Shard, blockchain block.Blockchain) {
	// TODO: Will there be a scenario where a replica is already present for the shard?
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
	hyperdrive.shards[shard.Hash] = shard
	hyperdrive.shardHistory = append(hyperdrive.shardHistory, shard.Hash)
	if len(hyperdrive.shardHistory) > NumHistoricalShards {
		delete(hyperdrive.shardReplicas, hyperdrive.shardHistory[0])
		hyperdrive.shardHistory = hyperdrive.shardHistory[1:]
	}

	r.Init()
}

func (hyperdrive *hyperdrive) validateBlock(shardHash sig.Hash, signedBlock block.SignedBlock) bool {
	// Block cannot be nil
	if signedBlock.Block.Equal(block.Block{}) {
		return false
	}
	// Block time cannot be later than current time
	if signedBlock.Time.After(time.Now()) {
		return false
	}
	if signedBlock.Round < 0 || signedBlock.Height < 0 {
		return false
	}

	// Verify the signatory of the signedBlock
	signatory, err := hyperdrive.signer.Verify(signedBlock.Block.Header, signedBlock.Signature)
	if err != nil {
		return false
	}
	if !signatory.Equal(signedBlock.Signatory) {
		return false
	}
	if !hyperdrive.isSignatoryInShard(shardHash, signatory) {
		return false
	}

	return true
}

func (hyperdrive *hyperdrive) validatePolka(shardHash sig.Hash, polka block.Polka) bool {
	// Polka cannot be nil
	if polka.Equal(block.Polka{}) {
		return false
	}
	if polka.Round < 0 || polka.Height < 0 {
		return false
	}

	preVote := block.PreVote{
		Block:  polka.Block,
		Height: polka.Height,
		Round:  polka.Round,
	}

	data := []byte(preVote.String())
	for i, signature := range polka.Signatures {
		if !hyperdrive.verifySignature(shardHash, data, signature, polka.Signatories[i]) {
			return false
		}
	}

	if polka.Block != nil {
		return hyperdrive.validateBlock(shardHash, *polka.Block)
	}

	return true
}

func (hyperdrive *hyperdrive) verifySignature(shardHash sig.Hash, data []byte, signature sig.Signature, signatory sig.Signatory) bool {
	hashSum256 := sha3.Sum256(data)
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	verifiedSig, err := hyperdrive.signer.Verify(hash, signature)
	if err != nil || !verifiedSig.Equal(signatory) {
		// TODO: log the error
		return false
	}

	return hyperdrive.isSignatoryInShard(shardHash, verifiedSig)
}

func (hyperdrive *hyperdrive) isSignatoryInShard(shardHash sig.Hash, signatory sig.Signatory) bool {
	if shard, ok := hyperdrive.shards[shardHash]; ok {
		for _, sig := range shard.Signatories {
			if signatory.Equal(sig) {
				return true
			}
		}
	}
	return false
}
