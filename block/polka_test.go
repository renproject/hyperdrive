package block_test

import (
	mathRand "math/rand"
	"time"

	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("PolkaBuilder", func() {
	Context("when PreVotes are inserted", func() {
		Context("when the pre-condition checks fails for Insert()", func() {
			Context("when the height is different from the block's height", func() {
				It("should panic", func() {
					builder := NewPolkaBuilder()
					block := Block{Height: 0, Round: 0}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					prevote := NewPreVote(&signedBlock, 0, 1)
					signedPreVote, err := prevote.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(func() { builder.Insert(signedPreVote) }).Should(Panic())
				})
			})

			Context("when the round is different from the block's round", func() {
				It("should panic", func() {
					builder := NewPolkaBuilder()
					block := Block{Height: 0, Round: 0}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					prevote := NewPreVote(&signedBlock, 1, 0)
					signedPreVote, err := prevote.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(func() { builder.Insert(signedPreVote) }).Should(Panic())
				})
			})
		})

		Context("when the pre-condition check fails for Polka()", func() {
			Context("when the consensus threshold is less than 1", func() {
				It("should panic", func() {
					builder := NewPolkaBuilder()
					Expect(func() { builder.Polka(0, 0) }).Should(Panic())
				})
			})

			Context("when too few pre-votes have been received", func() {
				It("should panic", func() {
					builder := NewPolkaBuilder()
					polka, polkaRound := builder.Polka(0, 11)
					Expect(polka).To(BeNil())
					Expect(polkaRound).To(BeNil())
				})
			})
		})

		Context("when less than the threshold of PreVotes is inserted", func() {
			Context("when PreVotes are inserted at the same height and the same round", func() {
				It("should never return a Polka", func() {
					builder := NewPolkaBuilder()
					for i := 0; i < 10; i++ {
						block := Block{Height: 0, Round: 0}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						prevote := NewPreVote(&signedBlock, 0, 0)
						signedPreVote, err := prevote.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(builder.Insert(signedPreVote)).To(BeTrue())
					}
					polka, polkaRound := builder.Polka(0, 11)
					Expect(polka).To(BeNil())
					Expect(polkaRound).To(BeNil())
				})
			})

			Context("when PreVotes are inserted at the same height and multiple rounds", func() {
				It("should never return a Polka", func() {
					builder := NewPolkaBuilder()
					for i := 0; i < 10; i++ {
						block := Block{Height: 0, Round: Round(i)}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						prevote := NewPreVote(&signedBlock, Round(i), 0)
						signedPreVote, err := prevote.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(builder.Insert(signedPreVote)).To(BeTrue())
					}
					polka, polkaRound := builder.Polka(0, 9)
					Expect(polka).To(BeNil())
					Expect(polkaRound).To(BeNil())
				})
			})

			Context("when PreVotes are inserted at multiple heights and the same round", func() {
				It("should never return a Polka", func() {
					builder := NewPolkaBuilder()
					for i := 0; i < 10; i++ {
						block := Block{Height: Height(i), Round: 0}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						prevote := NewPreVote(&signedBlock, 0, Height(i))
						signedPreVote, err := prevote.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(builder.Insert(signedPreVote)).To(BeTrue())
					}
					polka, polkaRound := builder.Polka(0, 9)
					Expect(polka).To(BeNil())
					Expect(polkaRound).To(BeNil())
				})
			})

			Context("when PreVotes are inserted at multiple heights and multiple rounds", func() {
				It("should never return a Polka", func() {
					builder := NewPolkaBuilder()
					height := Height(mathRand.Intn(100))
					for i := 0; i < 10; i++ {
						height = Height(mathRand.Intn(100))
						round := Round(mathRand.Intn(100))
						block := Block{Height: height, Round: round}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						prevote := NewPreVote(&signedBlock, round, height)
						signedPreVote, err := prevote.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(builder.Insert(signedPreVote)).To(BeTrue())
					}
					polka, polkaRound := builder.Polka(height, 9)
					Expect(polka).To(BeNil())
					Expect(polkaRound).To(BeNil())
				})
			})

			Context("when PreVotes with the same signature are added multiple times", func() {
				It("should never return a Polka", func() {
					builder := NewPolkaBuilder()
					height := Height(mathRand.Intn(100))
					height = Height(mathRand.Intn(100))
					round := Round(mathRand.Intn(100))
					block := Block{Height: height, Round: round}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					prevote := NewPreVote(&signedBlock, round, height)
					signedPreVote, err := prevote.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())

					Expect(builder.Insert(signedPreVote)).To(BeTrue())
					Expect(builder.Insert(signedPreVote)).To(BeFalse())
				})
			})
		})

		Context("when the threshold of PreVotes is inserted at the same round", func() {
			Context("when PreVotes are inserted for the same block", func() {
				It("should always return a Polka for the same block", func() {
					builder := NewPolkaBuilder()
					height := Height(mathRand.Intn(10))
					round := Round(mathRand.Intn(100))
					block := Block{
						Height: height,
						Round:  round,
						Header: testutils.RandomHash(),
					}
					for i := 0; i < 10; i++ {
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						prevote := NewPreVote(&signedBlock, round, height)
						signedPreVote, err := prevote.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(builder.Insert(signedPreVote)).To(BeTrue())
					}
					polka, polkaRound := builder.Polka(height, 9)
					Expect(polkaRound).To(Equal(&round))
					Expect(polka.Block.Block).To(Equal(block))
				})
			})

			Context("when PreVotes are inserted for different blocks", func() {
				It("should return a Polka for a nil block", func() {
					builder := NewPolkaBuilder()
					height := Height(mathRand.Intn(10))
					round := Round(mathRand.Intn(100))

					for i := 0; i < 10; i++ {
						block := Block{
							Height: height,
							Round:  round,
							Header: testutils.RandomHash(),
						}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						prevote := NewPreVote(&signedBlock, round, height)
						signedPreVote, err := prevote.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(builder.Insert(signedPreVote)).To(BeTrue())
					}
					polka, polkaRound := builder.Polka(height, 9)
					Expect(polkaRound).To(Equal(&round))
					Expect(polka).To(BeNil())
				})
			})
		})

		Context("when the threshold of PreVotes is inserted at multiple rounds", func() {
			It("should always return a Polka at the latest round", func() {
				builder := NewPolkaBuilder()
				for j := 0; j < 10; j++ {
					for i := 0; i < 10; i++ {
						block := Block{Height: 1, Round: Round(i)}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						prevote := NewPreVote(&signedBlock, Round(i), 1)
						signedPreVote, err := prevote.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(builder.Insert(signedPreVote)).To(BeTrue())
					}
				}

				polka, polkaRound := builder.Polka(1, 10)
				Expect(polka.Round).To(Equal(*polkaRound))
				Expect(polka.Round).To(Equal(Round(9)))
			})
		})
	})

	Context("when SignedPreVote is converted to string format", func() {
		It("should return the correct string representation", func() {
			block := Block{Height: 1, Round: 1, Time: time.Unix(0, 0)}
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			prevote := NewPreVote(&signedBlock, 1, 1)
			signedPreVote, err := prevote.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(signedPreVote.String()).Should(ContainSubstring("SignedPreVote(PreVote(Height=1,Round=1,Block(Height=1,Round=1,Timestamp=0,TxHeader=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=,ParentHeader=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=,Header=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=)),Signature="))
		})
	})

	Context("when Polka is converted to string format", func() {
		It("should return the correct string representation", func() {
			block := Block{Height: 1, Round: 1}
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			polka := Polka{
				Block:       &signedBlock,
				Round:       0,
				Height:      0,
				Signatures:  testutils.RandomSignatures(10),
				Signatories: testutils.RandomSignatories(10),
			}
			Expect(polka.String()).Should(Equal("Polka(Height=0,Round=0,BlockHeader=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=)"))
		})
	})

	Context("when Polkas are compared", func() {
		It("should return true if both Polkas are equal", func() {
			block := Block{Height: 1, Round: 1}
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			polka := Polka{
				Block:       &signedBlock,
				Round:       0,
				Height:      0,
				Signatures:  testutils.RandomSignatures(10),
				Signatories: testutils.RandomSignatories(10),
			}
			newPolka := polka
			Expect(polka.Equal(&newPolka)).Should(BeTrue())
		})

		It("should return false if both Polkas are equal, but signatories are different", func() {
			block := Block{Height: 1, Round: 1}
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			polka := Polka{
				Block:       &signedBlock,
				Round:       0,
				Height:      0,
				Signatures:  testutils.RandomSignatures(10),
				Signatories: testutils.RandomSignatories(10),
			}
			newPolka := Polka{
				Block:       &signedBlock,
				Round:       0,
				Height:      0,
				Signatures:  testutils.RandomSignatures(10),
				Signatories: testutils.RandomSignatories(10),
			}
			Expect(polka.Equal(&newPolka)).Should(BeFalse())
		})

		It("should return false if both Polkas are equal, but one of the blocks is nil", func() {
			block := Block{Height: 1, Round: 1}
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			polka := Polka{
				Block:       &signedBlock,
				Round:       0,
				Height:      0,
				Signatures:  testutils.RandomSignatures(10),
				Signatories: testutils.RandomSignatories(10),
			}
			newPolka := Polka{
				Block:       nil,
				Round:       0,
				Height:      0,
				Signatures:  polka.Signatures,
				Signatories: polka.Signatories,
			}
			Expect(polka.Equal(&newPolka)).Should(BeFalse())
		})
	})

	Context("when Drop is called on a specific Height", func() {
		It("should remove all SignedPreVotes below the given Height", func() {
			builder := NewPolkaBuilder()
			for j := 0; j < 10; j++ {
				for i := 0; i < 10; i++ {
					block := Block{Height: 1, Round: Round(i)}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					prevote := NewPreVote(&signedBlock, Round(i), 1)
					signedPreVote, err := prevote.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(builder.Insert(signedPreVote)).To(BeTrue())
				}
			}

			block := Block{Height: 2, Round: Round(10)}
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			prevote := NewPreVote(&signedBlock, Round(10), 2)
			signedPreVote, err := prevote.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(builder.Insert(signedPreVote)).To(BeTrue())

			polka, polkaRound := builder.Polka(1, 10)
			Expect(polka.Round).To(Equal(*polkaRound))
			Expect(polka.Round).To(Equal(Round(9)))

			builder.Drop(2)

			polka, polkaRound = builder.Polka(1, 1)
			Expect(polkaRound).To(BeNil())
			Expect(polka).To(BeNil())

			polka, polkaRound = builder.Polka(2, 1)
			Expect(polka.Round).To(Equal(*polkaRound))
			Expect(polka.Round).To(Equal(Round(10)))
		})
	})
})
