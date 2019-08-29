package testutil

//
// type MockBlockStorage struct {
// 	mu *sync.RWMutex
// 	shards  map[replica.Shard]*MockBlockchain
// }
//
// func NewMockBlockStorage()replica.BlockStorage{
// 	return &MockBlockStorage{
// 		mu:     new(sync.RWMutex),
// 		shards: map[replica.Shard]*MockBlockchain{},
// 	}
// }
//
// func (m *MockBlockStorage) Blockchain(shard replica.Shard) process.Blockchain {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
//
// 	blockchain, ok := m.shards[shard]
// 	if !ok {
// 		m.shards[shard] = &MockBlockchain{
// 			mu:     new(sync.RWMutex),
// 			blocks: map[block.Height]block.Block{},
// 		}
// 		return m.shards[shard]
// 	}
// 	return blockchain
// }
//
// func (m *MockBlockStorage) LatestBlock(shard replica.Shard) block.Block {
// 	m.mu.RLock()
// 	defer m.mu.RLock()
//
// 	blockchain, ok := m.shards[shard]
// 	if !ok {
// 		return block.InvalidBlock
// 	}
// 	return m.latestBlockOfKind(blockchain, block.Standard)
//
// }
//
// func (m *MockBlockStorage) LatestBaseBlock(shard replica.Shard) block.Block {
// 	m.mu.RLock()
// 	defer m.mu.RLock()
//
// 	blockchain, ok := m.shards[shard]
// 	if !ok {
// 		return block.InvalidBlock
// 	}
//
// 	return m.latestBlockOfKind(blockchain, block.Base)
// }
//
// func (m *MockBlockStorage) latestBlockOfKind(blockchain *MockBlockchain, kind block.Kind)block.Block{
// 	blockchain.mu.RLock()
// 	defer blockchain.mu.RLock()
//
// 	h, b := block.Height(0), block.Block{}
// 	for height, block := range blockchain.blocks{
// 		if height > h && block.Header().Kind() == kind {
// 			b = block
// 		}
// 	}
// 	return b
// }
