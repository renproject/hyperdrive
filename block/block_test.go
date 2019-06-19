package block_test

import (
	"math/rand"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/testutils"
	"github.com/renproject/hyperdrive/tx"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("Block", func() {
	Context("when blockchain is empty", func() {
		It("should return Genesis values", func() {
			genesis := Genesis()
			genesisCommit := Commit{
				Polka: Polka{
					Block: &genesis,
				},
			}
			blockchain := NewBlockchain(NewMockBlockStore())
			Expect(blockchain.Height()).To(Equal(genesis.Height))
			head, ok := blockchain.Head()
			Expect(ok).To(BeFalse())
			Expect(head).To(Equal(genesisCommit))
			block, ok := blockchain.Block(Height(0))
			Expect(ok).To(BeFalse())
			Expect(block).To(Equal(genesisCommit))
		})

		Context("when valid commits are inserted", func() {
			It("should return latest block", func() {
				blockchain := NewBlockchain(NewMockBlockStore())
				header := sig.Hash{}
				for i := 0; i < 10; i++ {
					block := Block{Height: Height(i), Header: testutils.RandomHash()}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())

					header = signedBlock.Header

					commit := Commit{
						Polka: Polka{
							Block:       &signedBlock,
							Round:       Round(i),
							Height:      Height(i),
							Signatures:  testutils.RandomSignatures(10),
							Signatories: testutils.RandomSignatories(10),
						},
					}
					blockchain.Extend(commit)
				}

				Expect(blockchain.Height()).To(Equal(Height(9)))
				head, ok := blockchain.Head()
				Expect(ok).To(BeTrue())
				Expect(head.Polka.Block.Header).To(Equal(header))
			})

			It("should return block for a specific header", func() {
				blockchain := NewBlockchain(NewMockBlockStore())
				queryIndex := rand.Intn(10)
				var queryBlock Commit
				for i := 0; i < 10; i++ {
					block := Block{Height: Height(i), Header: testutils.RandomHash()}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					commit := Commit{
						Polka: Polka{
							Block:       &signedBlock,
							Round:       Round(i),
							Height:      Height(i),
							Signatures:  testutils.RandomSignatures(10),
							Signatories: testutils.RandomSignatories(10),
						},
					}
					if i == queryIndex {
						queryBlock = commit
					}
					blockchain.Extend(commit)
				}

				block, ok := blockchain.Block(queryBlock.Polka.Height)
				Expect(ok).To(BeTrue())
				Expect(block).To(Equal(queryBlock))
			})

			It("should return blocks for a given range", func() {
				blockchain := NewBlockchain(NewMockBlockStore())
				for i := 0; i < 10; i++ {
					block := Block{Height: Height(i), Header: testutils.RandomHash()}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					commit := Commit{
						Polka: Polka{
							Block:       &signedBlock,
							Round:       Round(i),
							Height:      Height(i),
							Signatures:  testutils.RandomSignatures(10),
							Signatories: testutils.RandomSignatories(10),
						},
					}
					blockchain.Extend(commit)
				}

				// NOTE: Block range is inclusive.
				blocks := blockchain.Blocks(0, 0)
				Expect(len(blocks)).To(Equal(1))
				blocks = blockchain.Blocks(0, 4)
				Expect(len(blocks)).To(Equal(5))
				blocks = blockchain.Blocks(5, 9)
				Expect(len(blocks)).To(Equal(5))
				blocks = blockchain.Blocks(10, 15)
				Expect(len(blocks)).To(Equal(0))
			})

			Context("when nil commits are inserted", func() {
				It("should not insert the block", func() {
					genesis := Genesis()
					genesisCommit := Commit{
						Polka: Polka{
							Block: &genesis,
						},
					}
					commit := Commit{
						Polka: Polka{
							Block:       nil,
							Round:       0,
							Height:      0,
							Signatures:  testutils.RandomSignatures(10),
							Signatories: testutils.RandomSignatories(10),
						},
					}

					blockchain := NewBlockchain(NewMockBlockStore())
					blockchain.Extend(commit)

					Expect(blockchain.Height()).To(Equal(genesis.Height))
					head, ok := blockchain.Head()
					Expect(ok).To(BeFalse())
					Expect(head).To(Equal(genesisCommit))
				})
			})
		})
	})

	Context("when a new block is generated", func() {
		It("should populate the block header", func() {
			block := New(1, Genesis().Header, []tx.Transaction{testutils.RandomTransaction(), testutils.RandomTransaction()})
			Expect(block.Header).NotTo(BeNil())
			Expect(block.Header).NotTo(Equal(sig.Hash{}))
		})
	})

	Context("when genesis block is generated", func() {
		It("should return an empty block", func() {
			genesis := Genesis()
			expectedGenesis := SignedBlock{
				Block: Block{
					Time:         time.Unix(0, 0),
					Height:       0,
					Header:       sig.Hash{},
					ParentHeader: sig.Hash{},
					Txs:          tx.Transactions{},
				},
				Signature: sig.Signature{},
				Signatory: sig.Signatory{},
			}
			Expect(genesis).To(Equal(expectedGenesis))
		})
	})

	Context("when a new propose block is generated", func() {
		It("should not error while signing", func() {
			block := New(1, Genesis().Header, []tx.Transaction{testutils.RandomTransaction(), testutils.RandomTransaction()})
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			propose := Propose{
				Block: signedBlock,
				Round: 1,
			}
			signedPropose, err := propose.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(signedPropose.Round).To(Equal(Round(1)))
		})
	})
})

type mockBlockStore struct {
	height Height

	blocks   map[Height]Commit
	blocksMu *sync.RWMutex
}

func NewMockBlockStore() BlockStore {
	return &mockBlockStore{
		Genesis().Height,
		map[Height]Commit{},
		new(sync.RWMutex),
	}
}

func (mockBlockStore *mockBlockStore) Extend(commit Commit) error {
	mockBlockStore.blocksMu.Lock()
	defer mockBlockStore.blocksMu.Unlock()

	if mockBlockStore.height < commit.Polka.Height {
		mockBlockStore.height = commit.Polka.Height
	}
	mockBlockStore.blocks[commit.Polka.Height] = commit
	return nil
}

func (mockBlockStore *mockBlockStore) Block(height Height) (Commit, error) {
	mockBlockStore.blocksMu.RLock()
	defer mockBlockStore.blocksMu.RUnlock()

	return mockBlockStore.blocks[height], nil
}
func (mockBlockStore *mockBlockStore) Head() (Commit, error) {
	return mockBlockStore.Block(mockBlockStore.height)
}
func (mockBlockStore *mockBlockStore) Height() (Height, error) {
	return mockBlockStore.height, nil
}
