package testutil_replica

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/id"
	"github.com/renproject/phi"
	"golang.org/x/crypto/sha3"
)

func RandomShard() replica.Shard {
	shard := replica.Shard{}
	_, err := rand.Read(shard[:])
	if err != nil {
		panic(fmt.Sprintf("cannot create random shard, err = %v", err))
	}
	return shard
}

type MockBlockIterator struct {
	store *MockPersistentStorage
}

func NewMockBlockIterator(store *MockPersistentStorage) replica.BlockIterator {
	return &MockBlockIterator{
		store: store,
	}
}

func (m *MockBlockIterator) NextBlock(kind block.Kind, height block.Height, shard replica.Shard) (block.Data, block.State) {
	blockchain := m.store.MockBlockchain(shard)
	state, ok := blockchain.StateAtHeight(height - 1)
	if !ok {
		return testutil.RandomBytesSlice(), nil
	}

	switch kind {
	case block.Standard:
		return testutil.RandomBytesSlice(), state
	default:
		panic("unknown block kind")
	}
}

type MockValidator struct {
	store *MockPersistentStorage
}

func NewMockValidator(store *MockPersistentStorage) replica.Validator {
	return &MockValidator{
		store: store,
	}
}

func (m *MockValidator) IsBlockValid(b block.Block, checkHistory bool, shard replica.Shard) bool {
	height := b.Header().Height()
	prevState := b.PreviousState()

	blockchain := m.store.MockBlockchain(shard)
	if !checkHistory {
		return true
	}

	state, ok := blockchain.StateAtHeight(height - 1)
	if !ok {
		return false
	}
	if !bytes.Equal(prevState, state) {
		return false
	}
	return true
}

type MockObserver struct {
	store *MockPersistentStorage
}

func NewMockObserver(store *MockPersistentStorage) replica.Observer {
	return &MockObserver{
		store: store,
	}
}

func (m MockObserver) DidCommitBlock(height block.Height, shard replica.Shard) {
	blockchain := m.store.MockBlockchain(shard)
	block, ok := blockchain.BlockAtHeight(height)
	if !ok {
		panic("DidCommitBlock should be called only when the block has been added to storage")
	}
	digest := sha3.Sum256(block.Data())
	blockchain.InsertBlockStatAtHeight(height, digest[:])

	// Insert executed state of the previous height
	prevBlock, ok := blockchain.BlockAtHeight(height - 1)
	if !ok {
		panic(fmt.Sprintf("cannot find block of height %v, %v", height-1, prevBlock))
	}
	blockchain.InsertBlockStatAtHeight(height-1, prevBlock.PreviousState())
}
type latestMessages struct {
	Height    block.Height
	Propose  replica.Message
	Prevote  replica.Message
	Precommit replica.Message
}

type MockBroadcaster struct {
	min, max int

	mu     *sync.RWMutex
	cons   map[id.Signatory]chan replica.Message
	active map[id.Signatory]bool

	cacheMu  *sync.Mutex
	cachedMessages map[id.Signatory]*latestMessages // keep tracking of the messages of latest height
}

func NewMockBroadcaster(keys []*ecdsa.PrivateKey, min, max int) *MockBroadcaster {
	cons := map[id.Signatory]chan replica.Message{}
	cachedMessages := map[id.Signatory]*latestMessages{}

	for _, key := range keys {
		sig := id.NewSignatory(key.PublicKey)
		messages := make(chan replica.Message, 128)
		cons[sig] = messages
		cachedMessages[sig] = &latestMessages{}
	}

	return &MockBroadcaster{
		min:            min,
		max:            max,

		mu:             new(sync.RWMutex),
		cons:           cons,
		active:         map[id.Signatory]bool{},

		cacheMu:         new(sync.Mutex),
		cachedMessages:  cachedMessages,
	}
}

func (m *MockBroadcaster) Broadcast(message replica.Message) {
	m.cacheMu.Lock()
	var sender id.Signatory
	switch msg := message.Message.(type) {
	case *process.Propose:
		sender = msg.Signatory()

		latest := m.cachedMessages[sender]
		if msg.Height() >= latest.Height{
			latest.Height = msg.Height()
			latest.Propose = message
		}
	case *process.Prevote:
		sender = msg.Signatory()

		latest := m.cachedMessages[sender]
		if msg.Height() >= latest.Height{
			latest.Height = msg.Height()
			latest.Prevote = message
		}
	case *process.Precommit:
		sender = msg.Signatory()

		latest := m.cachedMessages[sender]
		if msg.Height() >= latest.Height{
			latest.Height = msg.Height()
			latest.Precommit = message
		}
	}
	m.cacheMu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	// If the sender is offline, it cannot send message to other nodes
	if !m.active[sender] {
		return
	}

	// If the receiver is offline, it cannot receive any message from other nodes.
	phi.ParForAll(m.cons, func(sig id.Signatory) {
		if m.active[sig] {
			m.sendMessage(sig, message)
		} else {

		}
	})
}

func (m *MockBroadcaster) sendMessage(receiver id.Signatory, message replica.Message) {
	messages := m.cons[receiver]
	// Simulate the network latency
	time.Sleep(time.Duration(mrand.Intn(m.max-m.min)+m.min) * time.Millisecond)

	// Drop the message if the node is not online
	messages <- message
}

func (m *MockBroadcaster) saveMessage (receiver id.Signatory, message replica.Message) {

}

func (m *MockBroadcaster) Messages(sig id.Signatory) chan replica.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cons[sig]
}

func (m *MockBroadcaster) EnablePeer(sig id.Signatory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active[sig] = true

	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	for signatory, latest := range m.cachedMessages{
		if signatory.Equal(sig) || !m.active[sig]{
			continue
		}
		if latest.Propose.Message != nil {
			m.cons[sig] <- latest.Propose
		}
		if latest.Prevote.Message != nil {
			m.cons[sig] <- latest.Prevote
		}
		if latest.Precommit.Message != nil {
			m.cons[sig] <- latest.Precommit
		}
	}
}

func (m *MockBroadcaster) DisablePeer(sig id.Signatory) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.active[sig] = false
}
