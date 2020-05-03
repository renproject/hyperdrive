package replica

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/abi"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/schedule"
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

func (shard Shard) SizeHint() int {
	return 32
}

func (shard Shard) Marshal(w io.Writer, m int) (int, error) {
	return abi.Bytes32(shard).Marshal(w, m)
}

func (shard *Shard) Unmarshal(r io.Reader, m int) (int, error) {
	return (*abi.Bytes32)(shard).Unmarshal(r, m)
}

type Messages []Message

// A Message sent/received by a Replica is composed of a Shard and the
// underlying `process.Message` data. It is expected that a Replica will sign
// the underlying `process.Message` data before sending the Message.
type Message struct {
	Message   process.Message
	Shard     Shard
	Signature id.Signature
}

// ProcessStorage saves and restores `process.State` to persistent memory. This
// guarantees that in the event of an unexpected shutdown, the Replica will only
// drop the `process.Message` that was currently being handling.
type ProcessStorage interface {
	SaveState(state *process.State, shard Shard)
	RestoreState(state *process.State, shard Shard)
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

	// MaxMessageQueueSize is the maximum number of "future" messages that the
	// Replica will buffer in memory.
	MaxMessageQueueSize int
}

func (options *Options) setZerosToDefaults() {
	if options.Logger == nil {
		options.Logger = logrus.StandardLogger()
	}
	if options.BackOffExp == 0 {
		options.BackOffExp = 1.6
	}
	if options.BackOffBase == time.Duration(0) {
		options.BackOffBase = 20 * time.Second
	}
	if options.BackOffMax == time.Duration(0) {
		options.BackOffMax = 5 * time.Minute
	}
	if options.MaxMessageQueueSize == 0 {
		options.MaxMessageQueueSize = 512
	}
}

type Replicas []Replica

// A Replica represents one Process in a replicated state machine that is bound
// to a specific Shard. It signs Messages before sending them to other Replicas,
// and verifies Messages before accepting them from other Replicas.
type Replica struct {
	options        Options
	shard          Shard
	p              *process.Process
	numSignatories int
	blockStorage   BlockStorage

	scheduler schedule.Scheduler
	rebaser   *shardRebaser
	cache     baseBlockCache

	messageQueue MessageQueue
}

func New(options Options, pStorage ProcessStorage, blockStorage BlockStorage, blockIterator BlockIterator, validator Validator, observer Observer, broadcaster Broadcaster, scheduler schedule.Scheduler, catcher process.Catcher, shard Shard, privKey ecdsa.PrivateKey) Replica {
	options.setZerosToDefaults()
	latestBase := blockStorage.LatestBaseBlock(shard)
	numSignatories := len(latestBase.Header().Signatories())
	if numSignatories%3 != 1 || numSignatories < 4 {
		panic(fmt.Errorf("invariant violation: number of nodes needs to be 3f +1, got %v", numSignatories))
	}
	shardRebaser := newShardRebaser(scheduler, blockStorage, blockIterator, validator, observer, shard, numSignatories)

	// Create a Process in the default state and then restore it
	p := process.New(
		options.Logger,
		id.NewSignatory(privKey.PublicKey),
		blockStorage.Blockchain(shard),
		process.DefaultState((numSignatories-1)/3),
		newSaveRestorer(pStorage, shard),
		shardRebaser,
		shardRebaser,
		shardRebaser,
		newSigner(broadcaster, shard, privKey),
		scheduler,
		newBackOffTimer(options.BackOffExp, options.BackOffBase, options.BackOffMax),
		catcher,
	)
	p.Restore()

	return Replica{
		options:        options,
		shard:          shard,
		p:              p,
		numSignatories: numSignatories,
		blockStorage:   blockStorage,

		scheduler: scheduler,
		rebaser:   shardRebaser,
		cache:     newBaseBlockCache(latestBase),

		messageQueue: NewMessageQueue(options.MaxMessageQueueSize),
	}
}

func (replica *Replica) Start() {
	replica.p.Start()
}

func (replica *Replica) HandleMessage(m Message) {
	// Check that Message is from our shard. If it is not, then there is no
	// point processing the message.
	if !replica.shard.Equal(m.Shard) {
		replica.options.Logger.Warnf("bad message: expected shard=%v, got shard=%v", replica.shard, m.Shard)
		return
	}

	// Ignore non-Resync messages from heights that the process has already
	// progressed through. Messages at these earlier heights have no affect on
	// consensus, and so there is no point wasting time processing them.
	if m.Message.Height() < replica.p.CurrentHeight() {
		if _, ok := m.Message.(*process.Resync); !ok {
			replica.options.Logger.Debugf("ignore message: expected height>=%v, got height=%v", replica.p.CurrentHeight(), m.Message.Height())
			return
		}
		// Fall-through to the remaining logic.
	}

	// Check that the Message sender is from our Shard (this can be a moderately
	// expensive operation, so we cache the result until a new `block.Base` is
	// detected)
	replica.cache.fillBaseBlock(replica.blockStorage.LatestBaseBlock(replica.shard))
	if !replica.cache.signatoryInBaseBlock(m.Message.Signatory()) {
		return
	}
	if err := replica.verifySignedMessage(m); err != nil {
		replica.options.Logger.Warnf("bad message: unverified: %v", err)
		return
	}

	// Resync messages can be handled immediately, as long as they are not from
	// a future height and their timestamps do not differ greatly from the
	// current time.
	if m.Message.Type() == process.ResyncMessageType {
		if m.Message.Height() > replica.p.CurrentHeight() {
			// We cannot respond to resync messages from future heights with
			// anything that is useful, so we ignore it.
			replica.options.Logger.Debugf("ignore message: resync height=%v compared to current height=%v", m.Message.Height(), replica.p.CurrentHeight())
			return
		}
		// Filter Resync messages by timestamp. If they're too old, or too far
		// in the future, then ignore them. The total window of time is 20
		// seconds, approximately the latency expected for globally distributed
		// message passing.
		now := block.Timestamp(time.Now().Unix())
		timestamp := m.Message.(*process.Resync).Timestamp()
		delta := now - timestamp
		if delta < 0 {
			delta = -delta
		}
		if delta > 10 {
			replica.options.Logger.Debugf("ignore message: resync timestamp=%v compared to now=%v", timestamp, now)
			return
		}
		replica.p.HandleMessage(m.Message)
		return
	}

	// Make sure that the Process state gets saved. We do this here
	// because Resync cannot cause state changes, so there is no
	// reason to save after handling a Resync message.
	defer replica.p.Save()

	// Messages from the current height can be handled immediately.
	if m.Message.Height() == replica.p.CurrentHeight() {
		replica.p.HandleMessage(m.Message)
	}

	// Messages from the future must be put into the height-ordered message
	// queue.
	if m.Message.Height() > replica.p.CurrentHeight() {
		if m.Message.Type() != process.ProposeMessageType {
			// We only want to queue non-Propose messages, because Propose messages
			// can have a LatestCommit message to fast-forward the process.
			replica.messageQueue.Push(m.Message)
		} else {
			// If the Propose is at a future height, then we need to make sure
			// that no base blocks have been missed. Otherwise, reject the
			// Propose, and wait until the appropriate one has been seen.
			baseBlockHash := replica.blockStorage.LatestBaseBlock(m.Shard).Hash()
			blockHash := m.Message.BlockHash()
			numMissingBaseBlocks := replica.rebaser.blockIterator.MissedBaseBlocksInRange(baseBlockHash, blockHash)
			if numMissingBaseBlocks == 0 {
				// If we have missed a base block, we drop the Propose. The
				// Propose that justifies the next base block will eventually be
				// seen by this Replica and we can begin accepting Proposes from
				// the new base.

				// In this condition, we haven't missed any base blocks, so we
				// can proceed as usual.
				replica.p.HandleMessage(m.Message)
			}
		}
	}

	queued := replica.messageQueue.PopUntil(replica.p.CurrentHeight())
	for queued != nil && len(queued) > 0 {
		baseBlockHash := replica.blockStorage.LatestBaseBlock(m.Shard).Hash()
		for _, message := range queued {
			// We need to make sure that no base blocks have been missed while
			// the message was sitting on the queue. If we have missed a base
			// block, we drop the message.
			blockHash := message.BlockHash()
			numMissingBaseBlocks := replica.rebaser.blockIterator.MissedBaseBlocksInRange(baseBlockHash, blockHash)
			if numMissingBaseBlocks == 0 {
				// Otherwise, we handle the Message. After all Messages that can
				// be handled have been handled, this function will end, and the
				// Process will be saved. This protects the Process from
				// crashing part way through handling a Message and ending up in
				// a partially saved State caused by "saving on the go". We
				// could save between each message, but this would have a large
				// performance footprint (and is ultimately unnecessary, because
				// we do not expect crashes).
				replica.p.HandleMessage(message)
			}
		}
		queued = replica.messageQueue.PopUntil(replica.p.CurrentHeight())
	}
}

func (replica *Replica) Rebase(sigs id.Signatories) {
	if len(sigs) != replica.numSignatories {
		panic(fmt.Errorf("invariant violation: number of signatories must not change: expected %v, got %v", replica.numSignatories, len(sigs)))
	}
	if len(sigs)%3 != 1 || len(sigs) < 4 {
		panic(fmt.Errorf("invariant violation: number of nodes needs to be 3f +1, got %v", len(sigs)))
	}
	replica.rebaser.rebase(sigs)
}

func (replica *Replica) verifySignedMessage(m Message) error {
	// Verify the Shard information
	mHash := m.SigHash()
	mPubKey, err := crypto.SigToPub(mHash[:], m.Signature[:])
	if err != nil {
		return fmt.Errorf("sigToPub: %v", err)
	}
	mSignatory := id.NewSignatory(*mPubKey)
	if !m.Message.Signatory().Equal(mSignatory) {
		return fmt.Errorf("bad signatory: expected signatory=%v, got signatory=%v", m.Message.Signatory(), mSignatory)
	}
	// Verify that the Message is actually signed by the claimed `id.Signatory`
	if err := process.Verify(m.Message); err != nil {
		return err
	}
	return nil
}

type saveRestorer struct {
	pStorage ProcessStorage
	shard    Shard
}

func newSaveRestorer(pStorage ProcessStorage, shard Shard) *saveRestorer {
	return &saveRestorer{
		pStorage: pStorage,
		shard:    shard,
	}
}

func (saveRestorer *saveRestorer) Save(state *process.State) {
	saveRestorer.pStorage.SaveState(state, saveRestorer.shard)
}
func (saveRestorer *saveRestorer) Restore(state *process.State) {
	saveRestorer.pStorage.RestoreState(state, saveRestorer.shard)
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
