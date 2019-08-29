package replica

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
	"github.com/sirupsen/logrus"
)

type Shards []Shard

// Shard uniquely identifies the Shard being maintained by the Replica.
type Shard [32]byte

// Equal compares one Shard with another.
func (shard Shard) Equal(other Shard) bool {
	return bytes.Equal(shard[:], other[:])
}

// String implements the `fmt.Stringer` interface.
func (shard Shard) String() string {
	return base64.RawStdEncoding.EncodeToString(shard[:])
}

type Messages []Message

// A Message sent/received by a Replica is composed of a Shard and the
// underlying `process.Message` data. It is expected that a Replica will sign
// the underlying `process.Message` data before sending the Message.
type Message struct {
	Message process.Message `json:"message"`
	Shard   Shard           `json:"shard"`
}

// ProcessStorage saves and restores `process.State` to persistent memory. This
// guarantess that in the event of an unexpected shutdown, the Replica will only
// drop the `process.Message` that was currently being handling.
type ProcessStorage interface {
	SaveProcess(p process.Process, shard Shard)
	RestoreProcess(p *process.Process, shard Shard)
}

// Options define a set of properties that can be used to parameterise the
// Replica and its behaviour.
type Options struct {
	// Logging
	Logger logrus.FieldLogger

	// Timeout options for proposing, prevoting, and precommiting
	BackOffExp  float64
	BackOffBase time.Duration
	BackOffMax  time.Duration
}

type Replicas []Replica

// A Replica represents one Process in a replicated state machine that is bound
// to a specific Shard. It signs Messages before sending them to other Replicas,
// and verifies Messages before accepting them from other Replicas.
type Replica struct {
	options      Options
	shard        Shard
	p            process.Process
	pStorage     ProcessStorage
	blockStorage BlockStorage
	cache        baseBlockCache
}

func New(options Options, pStorage ProcessStorage, blockStorage BlockStorage, blockIterator BlockIterator, validator Validator, observer Observer, broadcaster Broadcaster, shard Shard, privKey ecdsa.PrivateKey) Replica {
	shardRebaser := newShardRebaser(blockStorage, blockIterator, validator, observer, shard)
	latestBase := blockStorage.LatestBaseBlock()
	p := process.New(
		id.NewSignatory(privKey.PublicKey),
		blockStorage,
		// todo : fix f
		process.DefaultState(1),
		shardRebaser,
		shardRebaser,
		shardRebaser,
		newSigner(broadcaster, shard, privKey),
		NewRoundRobinScheduler(latestBase.Header().Signatories()),
		NewBackOffTimer(options.BackOffExp, options.BackOffBase, options.BackOffMax),
	)
	pStorage.RestoreProcess(&p, shard)
	return Replica{
		options:      options,
		shard:        shard,
		p:            p,
		pStorage:     pStorage,
		blockStorage: blockStorage,
		cache:        newBaseBlockCache(latestBase),
	}
}

func (replica *Replica) HandleMessage(m Message) {
	// Check that Message is from our Shard
	if !replica.shard.Equal(m.Shard) {
		replica.options.Logger.Warnf("bad message: expected shard=%v, got shard=%v", replica.shard, m.Shard)
		return
	}

	// Check that the Message sender is from our Shard (this can be a moderately
	// expensive operation, so we cache the result until a new `block.Base` is
	// detected)
	replica.cache.fillBaseBlock(replica.blockStorage.LatestBaseBlock())
	if !replica.cache.signatoryInBaseBlock(m.Message.Signatory()) {
		return
	}

	// Verify that the Message is actually signed by the claimed `id.Signatory`
	if err := process.Verify(m.Message); err != nil {
		replica.options.Logger.Warnf("bad message: unverified: %v", err)
		return
	}

	// Handle the underlying `process.Message` and immediately save the
	// `process.Process` afterwards to proected against unexpected crashes
	replica.p.HandleMessage(m.Message)
	replica.pStorage.SaveProcess(replica.p, replica.shard)
}

type baseBlockCache struct {
	lastBaseBlockHeight block.Height
	lastBaseBlockHash   id.Hash
	lastBaseBlockSigs   id.Signatories
	sigsCache           map[id.Signatory]bool
}

func newBaseBlockCache(baseBlock block.Block) baseBlockCache {
	cache := baseBlockCache{}
	cache.fillBaseBlock(baseBlock)
	return cache
}

func (cache *baseBlockCache) fillBaseBlock(baseBlock block.Block) {
	if baseBlock.Header().Height() <= cache.lastBaseBlockHeight {
		return
	}
	if baseBlock.Hash().Equal(cache.lastBaseBlockHash) {
		return
	}
	cache.lastBaseBlockHeight = baseBlock.Header().Height()
	cache.lastBaseBlockHash = baseBlock.Hash()
	cache.lastBaseBlockSigs = baseBlock.Header().Signatories()
	cache.sigsCache = map[id.Signatory]bool{}
}

func (cache *baseBlockCache) signatoryInBaseBlock(sig id.Signatory) bool {
	inBaseBlock, ok := cache.sigsCache[sig]
	if ok {
		return inBaseBlock
	}
	for _, baseBlockSig := range cache.lastBaseBlockSigs {
		if baseBlockSig.Equal(sig) {
			cache.sigsCache[sig] = true
			return true
		}
	}
	cache.sigsCache[sig] = false
	return false
}
