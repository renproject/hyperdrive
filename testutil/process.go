package testutil

import (
	"crypto/ecdsa"
	cRand "crypto/rand"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/id"
	"github.com/renproject/hyperdrive/process"
)

// RandomState returns a random `process.State`.
func RandomState() process.State {
	step := rand.Intn(3) + 1

	return process.State{
		CurrentHeight: block.Height(rand.Int63()),
		CurrentRound:  block.Round(rand.Int63()),
		CurrentStep:   process.Step(step),

		LockedBlock: RandomBlock(RandomBlockKind()),
		LockedRound: block.Round(rand.Int63()),
		ValidBlock:  RandomBlock(RandomBlockKind()),
		ValidRound:  block.Round(rand.Int63()),
	}
}

func RandomSignedMessage(t reflect.Type) process.Message {
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

func RandomMessage(t reflect.Type) process.Message {
	switch t {
	case reflect.TypeOf(process.Propose{}):
		return RandomPropose()
	case reflect.TypeOf(process.Prevote{}):
		return RandomPrevote()
	case reflect.TypeOf(process.Precommit{}):
		return RandomPrecommit()
	default:
		panic("unknown message type")
	}
}

func RandomMessageWithHeightAndRound(height block.Height, round block.Round, t reflect.Type) process.Message {
	var msg process.Message
	switch t {
	case reflect.TypeOf(process.Propose{}):
		validRound := block.Round(rand.Int63())
		block := RandomBlock(RandomBlockKind())
		msg = process.NewPropose(height, round, block, validRound)
	case reflect.TypeOf(process.Prevote{}):
		hash := RandomHash()
		msg = process.NewPrevote(height, round, hash)
	case reflect.TypeOf(process.Precommit{}):
		hash := RandomHash()
		msg = process.NewPrecommit(height, round, hash)
	default:
		panic("unknown message type")
	}
	return msg
}

func RandomSingedMessageWithHeightAndRound(height block.Height, round block.Round, t reflect.Type) process.Message {
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
	return process.NewPrevote(height, round, hash)
}

func RandomPrecommit() *process.Precommit {
	height := block.Height(rand.Int63())
	round := block.Round(rand.Int63())
	hash := RandomHash()
	return process.NewPrecommit(height, round, hash)
}

func RandomMessageType() reflect.Type {
	index := rand.Intn(3)
	switch index {
	case 0:
		return reflect.TypeOf(process.Propose{})
	case 1:
		return reflect.TypeOf(process.Prevote{})
	case 2:
		return reflect.TypeOf(process.Precommit{})
	default:
		panic("unexpect message type")
	}
}

func RandomInbox(f int, t reflect.Type) *process.Inbox {
	inbox := process.NewInbox(f, t)
	numMsgs := rand.Intn(100)
	for i := 0; i < numMsgs; i++ {
		inbox.Insert(RandomMessage(t))
	}
	return inbox
}

type ProcessOrigin struct {
	Signatory         id.Signatory
	Blockchain        process.Blockchain
	State             process.State
	BroadcastMessages chan process.Message

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
	messages := make(chan process.Message, 128)

	return ProcessOrigin{
		Signatory:         sig,
		Blockchain:        NewMockBlockchain(),
		State:             process.DefaultState(f),
		BroadcastMessages: messages,

		Proposer:    MockProposer{Key: privateKey},
		Validator:   NewMockValidator(true),
		Scheduler:   NewMockScheduler(sig),
		Broadcaster: NewMockBroadcaster(messages),
		Timer:       NewMockTimer(1 * time.Second),
		Observer:    MockObserver{},
	}
}

func (p ProcessOrigin) ToProcess() process.Process {
	return process.New(
		p.Signatory,
		p.Blockchain,
		p.State,
		p.Proposer,
		p.Validator,
		p.Observer,
		p.Broadcaster,
		p.Scheduler,
		p.Timer)
}

type MockBlockchain struct {
	mu     *sync.RWMutex
	blocks map[block.Height]block.Block
}

func NewMockBlockchain() process.Blockchain {
	return &MockBlockchain{
		mu:     new(sync.RWMutex),
		blocks: map[block.Height]block.Block{},
	}
}

func (bc *MockBlockchain) InsertBlockAtHeight(height block.Height, block block.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.blocks[height] = block
}

func (bc *MockBlockchain) BlockAtHeight(height block.Height) (block.Block, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	block, ok := bc.blocks[height]
	return block, ok
}

func (bc *MockBlockchain) BlockExistsAtHeight(height block.Height) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	_, ok := bc.blocks[height]
	return ok
}

type MockProposer struct {
	Key *ecdsa.PrivateKey
}

func (m MockProposer) BlockProposal(height block.Height, round block.Round) block.Block {
	header := RandomBlockHeaderJSON(RandomBlockKind())
	header.Height = height
	header.Round = round
	return block.New(header.ToBlockHeader(), RandomBytesSlice(), RandomBytesSlice())
}

type MockValidator struct {
	valid bool
}

func NewMockValidator(valid bool) process.Validator {
	return MockValidator{valid: valid}
}

func (m MockValidator) IsBlockValid(block.Block) bool {
	return m.valid
}

type MockObserver struct {
}

func (m MockObserver) DidCommitBlock(block.Height) {
}

type MockBroadcaster struct {
	messages chan<- process.Message
}

func NewMockBroadcaster(messages chan<- process.Message) process.Broadcaster {
	return &MockBroadcaster{
		messages: messages,
	}
}

func (m *MockBroadcaster) Broadcast(message process.Message) {
	m.messages <- message
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
