package testutil_replica

import (
	"fmt"
	"sync"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/surge"
)

type MockPersistentStorage struct {
	mu          *sync.RWMutex
	processes   map[replica.Shard][]byte
	blockchains map[replica.Shard]*testutil.MockBlockchain
}

func NewMockPersistentStorage(shards replica.Shards) *MockPersistentStorage {
	blockchains := map[replica.Shard]*testutil.MockBlockchain{}
	for _, shard := range shards {
		blockchains[shard] = testutil.NewMockBlockchain(nil)
	}
	return &MockPersistentStorage{
		mu:          new(sync.RWMutex),
		processes:   map[replica.Shard][]byte{},
		blockchains: blockchains,
	}
}

func (store *MockPersistentStorage) SaveState(state *process.State, shard replica.Shard) {
	data, err := surge.ToBinary(state)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal state: %v", err))

	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.processes[shard] = data
}

func (store *MockPersistentStorage) RestoreState(state *process.State, shard replica.Shard) {
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

func (store *MockPersistentStorage) Blockchain(shard replica.Shard) process.Blockchain {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, ok := store.blockchains[shard]
	if !ok {
		store.blockchains[shard] = testutil.NewMockBlockchain(nil)
	}
	return store.blockchains[shard]
}

func (store *MockPersistentStorage) MockBlockchain(shard replica.Shard) *testutil.MockBlockchain {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, ok := store.blockchains[shard]
	if !ok {
		store.blockchains[shard] = testutil.NewMockBlockchain(nil)
	}
	return store.blockchains[shard]
}

func (store *MockPersistentStorage) LatestBlock(shard replica.Shard) block.Block {
	store.mu.RLock()
	defer store.mu.RUnlock()

	blockchain := store.blockchains[shard]
	return blockchain.LatestBlock(block.Invalid)
}

func (store *MockPersistentStorage) LatestBaseBlock(shard replica.Shard) block.Block {
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
