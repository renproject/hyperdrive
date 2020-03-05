package testutil

import (
	"crypto/ecdsa"
	cRand "crypto/rand"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
	"github.com/sirupsen/logrus"
)

func RandomStep() process.Step {
	return process.Step(rand.Intn(3) + 1)
}

// RandomState returns a random `process.State`.
func RandomState() process.State {
	step := rand.Intn(3) + 1
	f := rand.Intn(100) + 1

	return process.State{
		CurrentHeight: block.Height(rand.Int63() + 1),
		CurrentRound:  block.Round(rand.Int63()),
		CurrentStep:   process.Step(step),

		LockedBlock: RandomBlock(RandomBlockKind()),
		LockedRound: block.Round(rand.Int63()),
		ValidBlock:  RandomBlock(RandomBlockKind()),
		ValidRound:  block.Round(rand.Int63()),

		Proposals:  process.NewInbox(f, process.ProposeMessageType),
		Prevotes:   process.NewInbox(f, process.PrevoteMessageType),
		Precommits: process.NewInbox(f, process.PrecommitMessageType),
	}
}

func RandomSignedMessage(t process.MessageType) process.Message {
	message := RandomMessage(t)
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
	if err != nil {
		panic(err)
	}
	if err := process.Sign(message, *privateKey); err != nil {
		panic(err)
	}

	return message
}

func RandomMessage(t process.MessageType) process.Message {
	switch t {
	case process.ProposeMessageType:
		return RandomPropose()
	case process.PrevoteMessageType:
		return RandomPrevote()
	case process.PrecommitMessageType:
		return RandomPrecommit()
	case process.ResyncMessageType:
		return RandomResync()
	default:
		panic("unknown message type")
	}
}

func RandomMessageWithHeightAndRound(height block.Height, round block.Round, t process.MessageType) process.Message {
	var msg process.Message
	switch t {
	case process.ProposeMessageType:
		validRound := block.Round(rand.Int63())
		block := RandomBlock(RandomBlockKind())
		msg = process.NewPropose(height, round, block, validRound)
	case process.PrevoteMessageType:
		hash := RandomHash()
		msg = process.NewPrevote(height, round, hash, nil)
	case process.PrecommitMessageType:
		hash := RandomHash()
		msg = process.NewPrecommit(height, round, hash)
	case process.ResyncMessageType:
		msg = process.NewResync(height, round)
	default:
		panic("unknown message type")
	}
	return msg
}

func RandomSingedMessageWithHeightAndRound(height block.Height, round block.Round, t process.MessageType) process.Message {
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
	if err != nil {
		panic(err)
	}
	msg := RandomMessageWithHeightAndRound(height, round, t)
	process.Sign(msg, *privateKey)
	return msg
}

func RandomPropose() *process.Propose {
	height := block.Height(rand.Int63())
	round := block.Round(rand.Int63())
	validRound := block.Round(rand.Int63())
	block := RandomBlock(RandomBlockKind())

	return process.NewPropose(height, round, block, validRound)
}

func RandomPrevote() *process.Prevote {
	height := block.Height(rand.Int63())
	round := block.Round(rand.Int63())
	hash := RandomHash()
	nilReasons := make(process.NilReasons)
	nilReasons["key1"] = []byte("val1")
	nilReasons["key2"] = []byte("val2")
	nilReasons["key3"] = []byte("val3")
	return process.NewPrevote(height, round, hash, nilReasons)
}

func RandomPrecommit() *process.Precommit {
	height := block.Height(rand.Int63())
	round := block.Round(rand.Int63())
	hash := RandomHash()
	return process.NewPrecommit(height, round, hash)
}

func RandomResync() *process.Resync {
	height := block.Height(rand.Int63())
	round := block.Round(rand.Int63())
	return process.NewResync(height, round)
}

func RandomMessageType() process.MessageType {
	index := rand.Intn(4)
	switch index {
	case 0:
		return process.ProposeMessageType
	case 1:
		return process.PrevoteMessageType
	case 2:
		return process.PrecommitMessageType
	case 3:
		return process.ResyncMessageType
	default:
		panic("unexpect message type")
	}
}

func RandomInbox(f int, t process.MessageType) *process.Inbox {
	inbox := process.NewInbox(f, t)
	numMsgs := rand.Intn(100)
	for i := 0; i < numMsgs; i++ {
		inbox.Insert(RandomMessage(t))
	}
	return inbox
}

type ProcessOrigin struct {
	PrivateKey        *ecdsa.PrivateKey
	Signatory         id.Signatory
	Blockchain        process.Blockchain
	State             process.State
	BroadcastMessages chan process.Message
	CastMessages      chan process.Message

	Proposer    process.Proposer
	Validator   process.Validator
	Scheduler   process.Scheduler
	Broadcaster process.Broadcaster
	Timer       process.Timer
	Observer    process.Observer
}

func NewProcessOrigin(f int) ProcessOrigin {
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
	if err != nil {
		panic(err)
	}
	sig := id.NewSignatory(privateKey.PublicKey)
	broadcastMessages := make(chan process.Message, 128)
	castMessages := make(chan process.Message, 128)
	signatories := make(id.Signatories, f)
	for i := range signatories {
		key, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
		if err != nil {
			panic(err)
		}
		signatories[i] = id.NewSignatory(key.PublicKey)
	}
	signatories[0] = sig

	return ProcessOrigin{
		PrivateKey:        privateKey,
		Signatory:         sig,
		Blockchain:        NewMockBlockchain(signatories),
		State:             process.DefaultState(f),
		BroadcastMessages: broadcastMessages,
		CastMessages:      castMessages,

		Proposer:    NewMockProposer(privateKey),
		Validator:   NewMockValidator(nil),
		Scheduler:   NewMockScheduler(sig),
		Broadcaster: NewMockBroadcaster(broadcastMessages, castMessages),
		Timer:       NewMockTimer(1 * time.Second),
		Observer:    MockObserver{},
	}
}

func (p ProcessOrigin) ToProcess() *process.Process {
	return process.New(
		logrus.StandardLogger(),
		p.Signatory,
		p.Blockchain,
		p.State,
		p.Proposer,
		p.Validator,
		p.Observer,
		p.Broadcaster,
		p.Scheduler,
		p.Timer,
	)
}

type MockBlockchain struct {
	mu     *sync.RWMutex
	blocks map[block.Height]block.Block
	states map[block.Height]block.State
}

func NewMockBlockchain(signatories id.Signatories) *MockBlockchain {
	blocks := map[block.Height]block.Block{}
	states := map[block.Height]block.State{}
	if signatories != nil {
		genesisblock := GenesisBlock(signatories)
		blocks[0] = genesisblock
		states[0] = block.State{}
	}

	return &MockBlockchain{
		mu:     new(sync.RWMutex),
		blocks: blocks,
		states: states,
	}
}

func (bc *MockBlockchain) InsertBlockAtHeight(height block.Height, block block.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.blocks[height] = block
}

func (bc *MockBlockchain) InsertBlockStatAtHeight(height block.Height, state block.State) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.states[height] = state
}

func (bc *MockBlockchain) BlockAtHeight(height block.Height) (block.Block, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	block, ok := bc.blocks[height]
	return block, ok
}

func (bc *MockBlockchain) StateAtHeight(height block.Height) (block.State, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	state, ok := bc.states[height]
	return state, ok
}

func (bc *MockBlockchain) BlockExistsAtHeight(height block.Height) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	_, ok := bc.blocks[height]
	return ok
}

func (bc *MockBlockchain) LatestBlock(kind block.Kind) block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	h, b := block.Height(-1), block.Block{}
	for height, blk := range bc.blocks {
		if height > h {
			if blk.Header().Kind() == kind || kind == block.Invalid {
				b = blk
				h = height
			}
		}
	}

	return b
}

type MockProposer struct {
	Key *ecdsa.PrivateKey
}

func NewMockProposer(key *ecdsa.PrivateKey) process.Proposer {
	return &MockProposer{Key: key}
}

func (m *MockProposer) BlockProposal(height block.Height, round block.Round) block.Block {
	header := RandomBlockHeaderJSON(RandomBlockKind())
	header.Height = height
	header.Round = round
	return block.New(header.ToBlockHeader(), RandomBytesSlice(), RandomBytesSlice(), RandomBytesSlice())
}

type MockValidator struct {
	valid error
}

func NewMockValidator(valid error) process.Validator {
	return MockValidator{valid: valid}
}

func (m MockValidator) IsBlockValid(block.Block, bool) (process.NilReasons, error) {
	return nil, m.valid
}

type MockObserver struct {
}

func (m MockObserver) DidCommitBlock(block.Height) {
}

func (m MockObserver) DidReceiveSufficientNilPrevotes(process.Messages, int) {
}

type MockBroadcaster struct {
	broadcastMessages chan<- process.Message
	castMessages      chan<- process.Message
}

func NewMockBroadcaster(broadcastMessages, castMessages chan<- process.Message) process.Broadcaster {
	return &MockBroadcaster{
		broadcastMessages: broadcastMessages,
		castMessages:      castMessages,
	}
}

func (m *MockBroadcaster) Broadcast(message process.Message) {
	m.broadcastMessages <- message
}

func (m *MockBroadcaster) Cast(to id.Signatory, message process.Message) {
	m.castMessages <- message
}

type MockScheduler struct {
	sig id.Signatory
}

func NewMockScheduler(sig id.Signatory) process.Scheduler {
	return &MockScheduler{sig: sig}
}

func (m *MockScheduler) Schedule(block.Height, block.Round) id.Signatory {
	return m.sig
}

type MockTimer struct {
	timeout time.Duration
}

func NewMockTimer(timeout time.Duration) process.Timer {
	return &MockTimer{
		timeout: timeout,
	}
}

func (timer *MockTimer) Timeout(step process.Step, round block.Round) time.Duration {
	return timer.timeout
}

func GetStateFromProcess(p *process.Process, f int) process.State {
	data, err := p.MarshalBinary()
	if err != nil {
		panic(err)
	}
	state := process.DefaultState(f)
	if err := state.UnmarshalBinary(data); err != nil {
		panic(err)
	}
	return state
}

func GenesisBlock(signatories id.Signatories) block.Block {
	header := block.NewHeader(
		block.Base,
		id.Hash{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		id.Hash{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		id.Hash{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		id.Hash{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		id.Hash{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		0, 0, 0, signatories,
	)
	return block.New(header, nil, nil, nil)
}
