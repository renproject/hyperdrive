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

var _ = XDescribe("Block", func() {
	Context("when blockchain is empty", func() {
		It("should return Genesis values", func() {
			genesis := Genesis()

			var blockchain Blockchain
			Expect(blockchain.Height()).To(Equal(genesis.Height))
			Expect(blockchain.Head()).To(Equal(genesis))
			block, ok := blockchain.Block(Height(0))
			Expect(ok).To(BeTrue())
			Expect(block).To(Equal(genesis))
		})

		Context("when valid commits are inserted", func() {
			It("should return latest block", func() {
				var blockchain Blockchain
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
				Expect( blockchain.Head().Header).To(Equal(header))
			})

			It("should return block for a specific header", func() {
				var blockchain Blockchain
				queryIndex := rand.Intn(10)
				queryBlock := Genesis()
				for i := 0; i < 10; i++ {
					block := Block{Height: Height(i), Header: testutils.RandomHash()}
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

			It("should return blocks for a given range", func() {
				var blockchain Blockchain
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

				blocks := blockchain.Blocks(0, 10)
				Expect(len(blocks)).To(Equal(10))

				blocks = blockchain.Blocks(10, 15)
				Expect(len(blocks)).To(Equal(0))
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

					var blockchain Blockchain
					blockchain.Extend(commit)

					Expect(blockchain.Height()).To(Equal(genesis.Height))
					Expect(blockchain.Head()).To(Equal(genesis))
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
