package replica

import (
	"crypto/ecdsa"
	cRand "crypto/rand"
	"math/rand"
	"testing/quick"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/testutil"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
)

var _ = Describe("shardRebaser", func() {

	commitBlock := func(storage BlockStorage, shard Shard, block block.Block) {
		bc := storage.Blockchain(shard)
		bc.InsertBlockAtHeight(block.Header().Height(), block)
	}

	Context("initializing a new shardRebaser", func() {
		It("should implements the process.Proposer", func() {
			test := func(shard Shard) bool {
				store, initHeight, _ := initStorage(shard)
				iter := mockBlockIterator{}
				rebaser := newShardRebaser(store, iter, nil, nil, shard)

				parent := store.LatestBlock(shard)
				base := store.LatestBaseBlock(shard)
				round := RandomRound()
				b := rebaser.BlockProposal(initHeight+1, round)

				Expect(b.Header().Kind()).Should(Equal(block.Standard))
				Expect(b.Header().ParentHash().Equal(parent.Hash())).Should(BeTrue())
				Expect(b.Header().BaseHash().Equal(base.Hash())).Should(BeTrue())
				Expect(b.Header().Height()).Should(Equal(initHeight + 1))
				Expect(b.Header().Round()).Should(Equal(round))
				Expect(b.Header().Signatories()).Should(BeNil())
				return true
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})

		It("should implements the process.Validator", func() {
			test := func(shard Shard) bool {
				store, initHeight, _ := initStorage(shard)
				iter := mockBlockIterator{}
				validator := newMockValidator(true)
				rebaser := newShardRebaser(store, iter, validator, nil, shard)

				// Generate a valid propose block.
				parent := store.LatestBlock(shard)
				base := store.LatestBaseBlock(shard)
				header := RandomBlockHeaderJSON(block.Standard)
				header.Height = initHeight + 1
				header.BaseHash = base.Hash()
				header.ParentHash = parent.Hash()
				header.Timestamp = block.Timestamp(time.Now().Unix())
				proposedBlock := block.New(header.ToBlockHeader(), nil, nil)

				return rebaser.IsBlockValid(proposedBlock, true)
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})

		It("should implements the process.Observer", func() {
			test := func(shard Shard) bool {
				store, initHeight, _ := initStorage(shard)
				iter := mockBlockIterator{}
				observer := newMockObserver()
				rebaser := newShardRebaser(store, iter, nil, observer, shard)

				rebaser.DidCommitBlock(0)
				rebaser.DidCommitBlock(initHeight)

				return true
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})

	Context("when rebasing", func() {
		It("should be ready to receive a new rebase block", func() {
			test := func(shard Shard, sigs id.Signatories) bool {
				store, _, _ := initStorage(shard)
				iter := mockBlockIterator{}
				rebaser := newShardRebaser(store, iter, nil, nil, shard)

				rebaser.rebase(sigs)
				Expect(rebaser.expectedKind).Should(Equal(block.Rebase))
				Expect(rebaser.expectedRebaseSigs.Equal(sigs)).Should(BeTrue())

				// Should be able to handle multiple rebase calls
				rebaser.rebase(sigs)
				Expect(rebaser.expectedKind).Should(Equal(block.Rebase))
				Expect(rebaser.expectedRebaseSigs.Equal(sigs)).Should(BeTrue())

				return true
			}
			Expect(quick.Check(test, nil)).Should(Succeed())
		})

		It("should return a valid rebase block when proposing", func() {
			test := func(shard Shard, sigs id.Signatories) bool {
				if len(sigs) == 0 {
					return true
				}
				store, initHeight, _ := initStorage(shard)
				iter := mockBlockIterator{}
				rebaser := newShardRebaser(store, iter, nil, nil, shard)

				rebaser.rebase(sigs)
				parent := store.LatestBlock(shard)
				base := store.LatestBaseBlock(shard)

				round := RandomRound()
				rebaseBlock := rebaser.BlockProposal(initHeight+1, round)
				Expect(rebaseBlock.Header().Kind()).Should(Equal(block.Rebase))
				Expect(rebaseBlock.Header().ParentHash().Equal(parent.Hash())).Should(BeTrue())
				Expect(rebaseBlock.Header().BaseHash().Equal(base.Hash())).Should(BeTrue())
				Expect(rebaseBlock.Header().Height()).Should(Equal(initHeight + 1))
				Expect(rebaseBlock.Header().Round()).Should(Equal(round))
				Expect(rebaseBlock.Header().Signatories().Equal(sigs)).Should(BeTrue())

				commitBlock(store, shard, rebaseBlock)
				rebaser.DidCommitBlock(initHeight + 1)

				baseBlock := rebaser.BlockProposal(initHeight+2, round)
				Expect(baseBlock.Header().Kind()).Should(Equal(block.Base))
				Expect(baseBlock.Header().ParentHash().Equal(rebaseBlock.Hash())).Should(BeTrue())
				Expect(baseBlock.Header().BaseHash().Equal(base.Hash())).Should(BeTrue())
				Expect(baseBlock.Header().Height()).Should(Equal(initHeight + 2))
				Expect(baseBlock.Header().Round()).Should(Equal(round))
				Expect(baseBlock.Header().Signatories().Equal(sigs)).Should(BeTrue())

				return true
			}
			Expect(quick.Check(test, nil)).Should(Succeed())
		})

		It("should valid a block only if it's a rebase block", func() {
			test := func(shard Shard, sigs id.Signatories) bool {
				if len(sigs) == 0 {
					return true
				}
				store, initHeight, _ := initStorage(shard)
				iter := mockBlockIterator{}
				rebaser := newShardRebaser(store, iter, nil, nil, shard)
				rebaser.rebase(sigs)

				// Generate a valid rebase block.
				parent := store.LatestBlock(shard)
				base := store.LatestBaseBlock(shard)
				header := RandomBlockHeaderJSON(block.Rebase)
				header.Height = initHeight + 1
				header.BaseHash = base.Hash()
				header.ParentHash = parent.Hash()
				header.Timestamp = block.Timestamp(time.Now().Unix() - 1)
				header.Signatories = sigs
				rebaseBlock := block.New(header.ToBlockHeader(), nil, nil)
				Expect(rebaser.IsBlockValid(rebaseBlock, true)).Should(BeTrue())

				// After the block been committed
				commitBlock(store, shard, rebaseBlock)
				rebaser.DidCommitBlock(initHeight + 1)

				// Generate a valid base block.
				parent = rebaseBlock
				baseHeader := RandomBlockHeaderJSON(block.Base)
				baseHeader.Height = initHeight + 2
				baseHeader.BaseHash = base.Hash()
				baseHeader.ParentHash = parent.Hash()
				baseHeader.Timestamp = block.Timestamp(time.Now().Unix())
				baseHeader.Signatories = sigs
				baseBlock := block.New(baseHeader.ToBlockHeader(), nil, nil)

				return rebaser.IsBlockValid(baseBlock, true)
			}
			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})

	// Context("when validating a proposed block", func() {
	// 	It("should reject block which has unexpect kind", func() {
	//
	// 	})
	// })
})

func initStorage(shard Shard) (BlockStorage, block.Height, []*ecdsa.PrivateKey) {
	sigs := make(id.Signatories, 7)
	keys := make([]*ecdsa.PrivateKey, 7)
	for i := range sigs {
		privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
		if err != nil {
			panic(err)
		}
		keys[i] = privateKey
		sigs[i] = id.NewSignatory(privateKey.PublicKey)
	}
	store := newMockBlockStorage(sigs)
	initHeight := block.Height(rand.Intn(100))

	// Init the genesis block at height 0
	bc := store.Blockchain(shard)

	// Init standard blocks from block 1 to initHeight
	for i := 1; i <= int(initHeight); i++ {
		b := RandomBlock(block.Standard)
		bc.InsertBlockAtHeight(block.Height(i), b)
	}
	return store, initHeight, keys
}
