package block_test

import (
	"math/rand"
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

			blockchain := Blockchain{}
			Expect(blockchain.Height()).To(Equal(genesis.Height))
			Expect(blockchain.Round()).To(Equal(genesis.Round))
			head, ok := blockchain.Head()
			Expect(ok).To(BeFalse())
			Expect(head).To(Equal(genesis))
			block, ok := blockchain.Block(0)
			Expect(ok).To(BeFalse())
			Expect(block).To(Equal(genesis))
		})

		Context("when valid commits are inserted", func() {
			It("should return latest block", func() {
				blockchain := NewBlockchain()
				block := Block{}
				signedBlock := SignedBlock{}
				for i := 0; i < 10; i++ {
					block = Block{Height: Height(i), Round: Round(i), Header: testutils.RandomHash()}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err = block.Sign(signer)
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

				Expect(blockchain.Height()).To(Equal(Height(9)))
				Expect(blockchain.Round()).To(Equal(Round(9)))
				head, ok := blockchain.Head()
				Expect(ok).To(BeTrue())
				Expect(head).To(Equal(signedBlock))
			})

			It("should return block for a specific header", func() {
				blockchain := NewBlockchain()
				queryIndex := rand.Intn(10)
				queryBlock := Genesis()
				for i := 0; i < 10; i++ {
					block := Block{Height: Height(i), Round: Round(i), Header: testutils.RandomHash()}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					if i == queryIndex {
						queryBlock = signedBlock
					}
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

				block, ok := blockchain.Block(queryBlock.Height)
				Expect(ok).To(BeTrue())
				Expect(block).To(Equal(queryBlock))
			})

			Context("when nil commits are inserted", func() {
				It("should not insert the block", func() {
					genesis := Genesis()
					commit := Commit{
						Polka: Polka{
							Block:       nil,
							Round:       0,
							Height:      0,
							Signatures:  testutils.RandomSignatures(10),
							Signatories: testutils.RandomSignatories(10),
						},
					}

					blockchain := NewBlockchain()
					blockchain.Extend(commit)

					Expect(blockchain.Height()).To(Equal(genesis.Height))
					Expect(blockchain.Round()).To(Equal(genesis.Round))
					head, ok := blockchain.Head()
					Expect(ok).To(BeTrue())
					Expect(head).To(Equal(genesis))
				})
			})
		})
	})

	Context("when a new block is generated", func() {
		It("should populate the block header", func() {
			block := New(1, 1, Genesis().Header, tx.Transactions{testutils.RandomTransaction(), testutils.RandomTransaction()}, []Commit{})
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
					Round:        0,
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
})
