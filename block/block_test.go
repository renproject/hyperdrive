package block_test

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/renproject/hyperdrive/sig"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("Blockchain", func() {
	Context("when blockchain is empty", func() {
		It("should return Genesis values", func() {
			genesis := Genesis()

			blockchain := Blockchain{}
			Expect(blockchain.Height()).To(Equal(genesis.Height))
			Expect(blockchain.Round()).To(Equal(genesis.Round))
			head, ok := blockchain.Head()
			Expect(ok).To(BeFalse())
			Expect(head).To(Equal(genesis))
			block, ok := blockchain.Block(sig.Hash{})
			Expect(ok).To(BeFalse())
			Expect(block).To(Equal(genesis))
		})

		Context("when valid commits are inserted", func() {
			It("should return latest block", func() {
				blockchain := NewBlockchain()
				block := Block{}
				for i := 0; i < 10; i++ {
					block = Block{Height: Height(i), Round: Round(i), Header: randomHash()}
					commit := Commit{
						Polka: Polka{
							Block:       &block,
							Round:       Round(i),
							Height:      Height(i),
							Signatures:  randomSignatures(10),
							Signatories: randomSignatories(10),
						},
					}
					blockchain.Extend(commit)
				}

				Expect(blockchain.Height()).To(Equal(Height(9)))
				Expect(blockchain.Round()).To(Equal(Round(9)))
				head, ok := blockchain.Head()
				Expect(ok).To(BeTrue())
				Expect(head).To(Equal(block))
			})

			It("should return block for a specific header", func() {
				blockchain := NewBlockchain()
				queryIndex := rand.Intn(10)
				queryBlock := Genesis()
				for i := 0; i < 10; i++ {
					block := Block{Height: Height(i), Round: Round(i), Header: randomHash()}
					if i == queryIndex {
						fmt.Println(i)
						queryBlock = block
					}
					commit := Commit{
						Polka: Polka{
							Block:       &block,
							Round:       Round(i),
							Height:      Height(i),
							Signatures:  randomSignatures(10),
							Signatories: randomSignatories(10),
						},
					}
					blockchain.Extend(commit)
				}

				block, ok := blockchain.Block(queryBlock.Header)
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
							Signatures:  randomSignatures(10),
							Signatories: randomSignatories(10),
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

	Context("when genesis block is generated", func() {
		It("should create the correct genesis block", func() {
			genesis := Genesis()
			expectedGenesis := Block{
				Time:         time.Unix(0, 0),
				Round:        0,
				Height:       0,
				Header:       sig.Hash{},
				ParentHeader: sig.Hash{},
				Signature:    sig.Signature{},
				Signatory:    sig.Signatory{},
			}
			Expect(genesis).To(Equal(expectedGenesis))
		})
	})
})
