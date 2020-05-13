package replica

import (
	"sync"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/id"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
)

func TestReplica(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Replica Suite")
}

type mockBlockStorage struct {
	mu     *sync.RWMutex
	sigs   id.Signatories
	shards map[process.Shard]*MockBlockchain
}

func newMockBlockStorage(sigs id.Signatories) BlockStorage {
	return &mockBlockStorage{
		mu:     new(sync.RWMutex),
		sigs:   sigs,
		shards: map[process.Shard]*MockBlockchain{},
	}
}

func (m *mockBlockStorage) Blockchain(shard process.Shard) process.Blockchain {
	m.mu.Lock()
	defer m.mu.Unlock()

	blockchain, ok := m.shards[shard]
	if !ok {
		m.shards[shard] = NewMockBlockchain(m.sigs)
		return m.shards[shard]
	}
	return blockchain
}

func (m *mockBlockStorage) LatestBlock(shard process.Shard) block.Block {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blockchain, ok := m.shards[shard]
	if !ok {
		return block.InvalidBlock
	}

	return blockchain.LatestBlock(block.Invalid)
}

func (m *mockBlockStorage) LatestBaseBlock(shard process.Shard) block.Block {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blockchain, ok := m.shards[shard]
	if !ok {
		return block.InvalidBlock
	}

	return blockchain.LatestBlock(block.Base)
}

type mockBlockIterator struct {
}

func (m mockBlockIterator) NextBlock(kind block.Kind, height block.Height, shard process.Shard) (block.Txs, block.Plan, block.State) {
	return RandomBytesSlice(), RandomBytesSlice(), RandomBytesSlice()
}

func (m mockBlockIterator) MissedBaseBlocksInRange(begin, end id.Hash) int {
	return 0 // mockBlockIterator does not support rebasing.
}

type mockValidator struct {
	valid error
}

func (m mockValidator) IsBlockValid(block.Block, bool, process.Shard) (process.NilReasons, error) {
	return nil, m.valid
}

func newMockValidator(valid error) Validator {
	return mockValidator{valid: valid}
}

type mockObserver struct {
}

func newMockObserver() Observer {
	return mockObserver{}
}

func (m mockObserver) DidCommitBlock(block.Height, process.Shard) {
}
func (m mockObserver) IsSignatory(process.Shard) bool {
	return true
}
func (m mockObserver) DidReceiveSufficientNilPrevotes(process.Messages, int) {
}

type mockProcessStorage struct {
}

func (m mockProcessStorage) SaveState(state *process.State, shard process.Shard) {
}

func (m mockProcessStorage) RestoreState(state *process.State, shard process.Shard) {
}
