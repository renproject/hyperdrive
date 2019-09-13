package replica

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	Message process.Message
	Shard   Shard
}

func (m Message) MarshalJSON() ([]byte, error) {
	tmp := struct {
		MessageType process.MessageType `json:"type"`
		Message     process.Message     `json:"message"`
		Shard       Shard               `json:"shard"`
	}{
		MessageType: m.Message.Type(),
		Message:     m.Message,
		Shard:       m.Shard,
	}
	return json.Marshal(tmp)
}

func (m *Message) UnmarshalJSON(data []byte) error {
	tmp := struct {
		MessageType process.MessageType `json:"type"`
		Message     json.RawMessage     `json:"message"`
		Shard       Shard               `json:"shard"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	switch tmp.MessageType {
	case process.ProposeMessageType:
		propose := new(process.Propose)
		if err := propose.UnmarshalJSON(tmp.Message); err != nil {
			return err
		}
		m.Message = propose
	case process.PrevoteMessageType:
		prevote := new(process.Prevote)
		if err := prevote.UnmarshalJSON(tmp.Message); err != nil {
			return err
		}
		m.Message = prevote
	case process.PrecommitMessageType:
		precommit := new(process.Precommit)
		if err := precommit.UnmarshalJSON(tmp.Message); err != nil {
			return err
		}
		m.Message = precommit
	}
	m.Shard = tmp.Shard

	return nil
}

// ProcessStorage saves and restores `process.State` to persistent memory. This
// guarantess that in the event of an unexpected shutdown, the Replica will only
// drop the `process.Message` that was currently being handling.
type ProcessStorage interface {
	SaveProcess(p *process.Process, shard Shard)
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

func (options *Options) setZerosToDefaults() {
	if options.Logger == nil {
		options.Logger = logrus.StandardLogger()
	}
	if options.BackOffExp == 0 {
		options.BackOffExp = 1.6
	}
	if options.BackOffBase == time.Duration(0) {
		options.BackOffBase = 5 * time.Second
	}
	if options.BackOffMax == time.Duration(0) {
		options.BackOffMax = time.Minute
	}
}

type Replicas []Replica

// A Replica represents one Process in a replicated state machine that is bound
// to a specific Shard. It signs Messages before sending them to other Replicas,
// and verifies Messages before accepting them from other Replicas.
type Replica struct {
	options      Options
	shard        Shard
	p            *process.Process
	pStorage     ProcessStorage
	blockStorage BlockStorage

	scheduler *roundRobinScheduler
	rebaser   *shardRebaser
	cache     baseBlockCache

	messagesSinceLastSave int
}

func New(options Options, pStorage ProcessStorage, blockStorage BlockStorage, blockIterator BlockIterator, validator Validator, observer Observer, broadcaster Broadcaster, shard Shard, privKey ecdsa.PrivateKey) Replica {
	options.setZerosToDefaults()
	latestBase := blockStorage.LatestBaseBlock(shard)
	scheduler := newRoundRobinScheduler(latestBase.Header().Signatories())
	if len(latestBase.Header().Signatories())%3 != 1 {
		panic(fmt.Errorf("invariant violation: number of nodes needs to be 3f +1, got %v", len(latestBase.Header().Signatories())))
	}
	shardRebaser := newShardRebaser(blockStorage, blockIterator, validator, observer, shard)

	// Create a Process in the default state and then restore it
	p := process.New(
		options.Logger,
		id.NewSignatory(privKey.PublicKey),
		blockStorage.Blockchain(shard),
		process.DefaultState((len(latestBase.Header().Signatories())-1)/3),
		shardRebaser,
		shardRebaser,
		shardRebaser,
		newSigner(broadcaster, shard, privKey),
		scheduler,
		newBackOffTimer(options.BackOffExp, options.BackOffBase, options.BackOffMax),
	)
	pStorage.RestoreProcess(p, shard)

	return Replica{
		options:      options,
		shard:        shard,
		p:            p,
		pStorage:     pStorage,
		blockStorage: blockStorage,

		scheduler: scheduler,
		rebaser:   shardRebaser,
		cache:     newBaseBlockCache(latestBase),

		messagesSinceLastSave: 0,
	}
}

func (replica *Replica) Start() {
	replica.p.Start()
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
	replica.cache.fillBaseBlock(replica.blockStorage.LatestBaseBlock(replica.shard))
	if !replica.cache.signatoryInBaseBlock(m.Message.Signatory()) {
		return
	}

	// Verify that the Message is actually signed by the claimed `id.Signatory`
	if err := process.Verify(m.Message); err != nil {
		replica.options.Logger.Warnf("bad message: unverified: %v", err)
		return
	}

	// Handle the underlying `process.Message` and immediately save the
	// `process.Process` afterwards to protect against unexpected crashes
	replica.p.HandleMessage(m.Message)

	// Save process to storage every 10 messages
	replica.messagesSinceLastSave++
	if replica.messagesSinceLastSave >= 10 {
		replica.pStorage.SaveProcess(replica.p, replica.shard)
		replica.messagesSinceLastSave = 0
	}
}

func (replica *Replica) Rebase(sigs id.Signatories) {
	replica.scheduler.rebase(sigs)
	replica.rebaser.rebase(sigs)
}

type baseBlockCache struct {
	lastBaseBlockHeight block.Height
	lastBaseBlockHash   id.Hash
	lastBaseBlockSigs   id.Signatories
	sigsCache           map[id.Signatory]bool
}

func newBaseBlockCache(baseBlock block.Block) baseBlockCache {
	cache := baseBlockCache{
		lastBaseBlockHeight: -1,
		sigsCache:           map[id.Signatory]bool{},
	}
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
