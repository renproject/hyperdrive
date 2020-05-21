package testutil_replica

import (
	"fmt"
	"sync"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/surge"
)

type MockPersistentStorage struct {
	mu          *sync.RWMutex
	processes   map[process.Shard][]byte
	blockchains map[process.Shard]*testutil.MockBlockchain
}

func NewMockPersistentStorage(shards process.Shards) *MockPersistentStorage {
	blockchains := map[process.Shard]*testutil.MockBlockchain{}
	for _, shard := range shards {
		blockchains[shard] = testutil.NewMockBlockchain(nil)
	}
	return &MockPersistentStorage{
		mu:          new(sync.RWMutex),
		processes:   map[process.Shard][]byte{},
		blockchains: blockchains,
	}
}

func (store *MockPersistentStorage) SaveState(state *process.State, shard process.Shard) {
	data, err := surge.ToBinary(state)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal state: %v", err))

	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.processes[shard] = data
}

func (store *MockPersistentStorage) RestoreState(state *process.State, shard process.Shard) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	data, ok := store.processes[shard]
	if !ok {
		return
	}
	if err := surge.FromBinary(data, state); err != nil {
		panic(fmt.Sprintf("failed to unmarshal state: %v", err))
	}
}

func (store *MockPersistentStorage) Blockchain(shard process.Shard) process.Blockchain {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, ok := store.blockchains[shard]
	if !ok {
		store.blockchains[shard] = testutil.NewMockBlockchain(nil)
	}
	return store.blockchains[shard]
}

func (store *MockPersistentStorage) MockBlockchain(shard process.Shard) *testutil.MockBlockchain {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, ok := store.blockchains[shard]
	if !ok {
		store.blockchains[shard] = testutil.NewMockBlockchain(nil)
	}
	return store.blockchains[shard]
}

func (store *MockPersistentStorage) LatestBlock(shard process.Shard) block.Block {
	store.mu.RLock()
	defer store.mu.RUnlock()

	blockchain := store.blockchains[shard]
	return blockchain.LatestBlock(block.Invalid)
}

func (store *MockPersistentStorage) LatestBaseBlock(shard process.Shard) block.Block {
	store.mu.Lock()
	defer store.mu.Unlock()

	blockchain, ok := store.blockchains[shard]
	if !ok {
		return block.InvalidBlock
	}
	return blockchain.LatestBlock(block.Base)
}

func (store *MockPersistentStorage) Init(gb block.Block) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for _, bc := range store.blockchains {
		bc.InsertBlockAtHeight(block.Height(0), gb)
		bc.InsertBlockStateAtHeight(block.Height(0), nil)
	}
}
