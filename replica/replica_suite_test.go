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
	shards map[Shard]*MockBlockchain
}

func newMockBlockStorage(sigs id.Signatories) BlockStorage {
	return &mockBlockStorage{
		mu:     new(sync.RWMutex),
		sigs:   sigs,
		shards: map[Shard]*MockBlockchain{},
	}
}

func (m *mockBlockStorage) Blockchain(shard Shard) process.Blockchain {
	m.mu.Lock()
	defer m.mu.Unlock()

	blockchain, ok := m.shards[shard]
	if !ok {
		m.shards[shard] = NewMockBlockchain(m.sigs)
		return m.shards[shard]
	}
	return blockchain
}

func (m *mockBlockStorage) LatestBlock(shard Shard) block.Block {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blockchain, ok := m.shards[shard]
	if !ok {
		return block.InvalidBlock
	}

	return blockchain.LatestBlock(block.Invalid)
}

func (m *mockBlockStorage) LatestBaseBlock(shard Shard) block.Block {
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

func (m mockBlockIterator) NextBlock(kind block.Kind, height block.Height, shard Shard) (block.Txs, block.Plan, block.State) {
	return RandomBytesSlice(), RandomBytesSlice(), RandomBytesSlice()
}

type mockValidator struct {
	valid error
}

func (m mockValidator) IsBlockValid(block.Block, bool, Shard) (map[string]interface{}, error) {
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

func (m mockObserver) DidCommitBlock(block.Height, Shard) {
}
func (m mockObserver) IsSignatory(Shard) bool {
	return true
}
func (m mockObserver) ReceivedSufficientNilPrevotes(process.Messages, int) {
}

type mockProcessStorage struct {
}

func (m mockProcessStorage) SaveProcess(p *process.Process, shard Shard) {
}

func (m mockProcessStorage) RestoreProcess(p *process.Process, shard Shard) {
}
