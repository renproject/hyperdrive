package block_test

import (
	mathRand "math/rand"

	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("CommitBuilder", func() {
	Context("when PreCommmits are inserted", func() {
		Context("when the pre-condition checks fails for Insert()", func() {
			Context("when the height is different from the block's height", func() {
				It("should panic", func() {
					builder := CommitBuilder{}
					block := Block{
						Height: 1,
						Round:  0,
					}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())

					precommit := PreCommit{
						Polka: Polka{
							Block:  &signedBlock,
							Height: 0,
							Round:  0,
						},
					}
					signedPreCommit, err := precommit.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(func() { builder.Insert(signedPreCommit) }).Should(Panic())
				})
			})

			Context("when the round is different from the block's round", func() {
				It("should panic", func() {
					builder := CommitBuilder{}
					block := Block{
						Height: 0,
						Round:  1,
					}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					precommit := PreCommit{
						Polka: Polka{
							Block:  &signedBlock,
							Height: 0,
							Round:  0,
						},
					}
					signedPreCommit, err := precommit.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(func() { builder.Insert(signedPreCommit) }).Should(Panic())
				})
			})
		})

		Context("when the pre-condition check fails for Commit()", func() {
			Context("when the consensus threshold is less than 1", func() {
				It("should panic", func() {
					builder := CommitBuilder{}
					Expect(func() { builder.Commit(0, 0) }).Should(Panic())
				})
			})

			Context("when too few pre-votes have been received", func() {
				It("should panic", func() {
					builder := CommitBuilder{}
					_, ok := builder.Commit(0, 11)
					Expect(ok).To(BeFalse())
				})
			})
		})

		Context("when less than the threshold of PreCommits is inserted", func() {
			Context("when PreCommits are inserted at the same height and the same round", func() {
				It("should never return a Commit", func() {
					builder := CommitBuilder{}
					for i := 0; i < 10; i++ {
						block := Block{
							Height: 0,
							Round:  0,
						}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						precommit := PreCommit{
							Polka: Polka{
								Block:  &signedBlock,
								Height: 0,
								Round:  0,
							},
						}
						signedPreCommit, err := precommit.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						builder.Insert(signedPreCommit)
					}
					_, ok := builder.Commit(0, 11)
					Expect(ok).To(BeFalse())
				})
			})

			Context("when PreCommits are inserted at the same height and multiple rounds", func() {
				It("should never return a Commit", func() {
					builder := CommitBuilder{}
					for i := 0; i < 10; i++ {
						block := Block{
							Height: 0,
							Round:  Round(i),
						}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						precommit := PreCommit{
							Polka: Polka{
								Block:  &signedBlock,
								Height: 0,
								Round:  Round(i),
							},
						}
						signedPreCommit, err := precommit.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						builder.Insert(signedPreCommit)
					}
					_, ok := builder.Commit(0, 9)
					Expect(ok).To(BeFalse())
				})
			})

			Context("when PreCommits are inserted at multiple heights and the same round", func() {
				It("should never return a Commit", func() {
					builder := CommitBuilder{}
					for i := 0; i < 10; i++ {
						block := Block{
							Height: Height(i),
							Round:  0,
						}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						precommit := PreCommit{
							Polka: Polka{
								Block:  &signedBlock,
								Height: Height(i),
								Round:  0,
							},
						}
						signedPreCommit, err := precommit.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						builder.Insert(signedPreCommit)
					}
					_, ok := builder.Commit(0, 9)
					Expect(ok).To(BeFalse())
				})
			})

			Context("when PreCommits are inserted at multiple heights and multiple rounds", func() {
				It("should never return a Commit", func() {
					builder := CommitBuilder{}
					height := Height(mathRand.Intn(100))
					for i := 0; i < 10; i++ {
						height = Height(mathRand.Intn(100))
						round := Round(mathRand.Intn(100))
						block := Block{
							Height: height,
							Round:  round,
						}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						precommit := PreCommit{
							Polka: Polka{
								Block:  &signedBlock,
								Height: height,
								Round:  round,
							},
						}
						signedPreCommit, err := precommit.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						builder.Insert(signedPreCommit)
					}
					_, ok := builder.Commit(height, 9)
					Expect(ok).To(BeFalse())
				})
			})
		})

		Context("when the threshold of PreCommits is inserted at the same round", func() {
			Context("when PreCommits are inserted for the same block", func() {
				It("should always return a Commit for the same block", func() {
					builder := CommitBuilder{}
					height := Height(mathRand.Intn(10))
					round := Round(mathRand.Intn(100))

					block := Block{
						Height: height,
						Round:  round,
						Header: testutils.RandomHash(),
					}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())

					for i := 0; i < 10; i++ {
						precommit := PreCommit{
							Polka: Polka{
								Block:  &signedBlock,
								Height: height,
								Round:  round,
							},
						}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedPreCommit, err := precommit.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						builder.Insert(signedPreCommit)
					}
					commit, ok := builder.Commit(height, 9)
					Expect(ok).To(BeTrue())
					Expect(commit.Polka.Block).To(Equal(&signedBlock))
				})
			})

			Context("when PreCommits are inserted for different blocks", func() {
				It("should return a Commit for a nil block", func() {
					builder := CommitBuilder{}
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
						precommit := PreCommit{
							Polka: Polka{
								Block:  &signedBlock,
								Height: height,
								Round:  round,
							},
						}
						signedPreCommit, err := precommit.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						builder.Insert(signedPreCommit)
					}
					commit, ok := builder.Commit(height, 9)
					Expect(ok).To(BeTrue())
					Expect(commit.Polka.Block).To(BeNil())
				})
			})
		})

		Context("when the threshold of PreCommits is inserted at multiple rounds", func() {
			It("should always return a Commit at the latest round", func() {
				builder := CommitBuilder{}
				for j := 0; j < 10; j++ {
					for i := 0; i < 10; i++ {
						block := Block{
							Height: 1,
							Round:  Round(i),
						}
						signer, err := ecdsa.NewFromRandom()
						Expect(err).ShouldNot(HaveOccurred())
						signedBlock, err := block.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						precommit := PreCommit{
							Polka: Polka{
								Block:  &signedBlock,
								Height: 1,
								Round:  Round(i),
							},
						}
						signedPreCommit, err := precommit.Sign(signer)
						Expect(err).ShouldNot(HaveOccurred())
						builder.Insert(signedPreCommit)
					}
				}

				commit, ok := builder.Commit(1, 10)
				Expect(ok).To(BeTrue())
				Expect(commit.Polka.Round).To(Equal(Round(9)))
			})
		})

	})

	Context("when Commit is converted to string format", func() {
		It("should return the correct string representation", func() {
			block := Block{Height: 1, Round: 1}
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			commit := Commit{
				Polka: Polka{
					Block:       &signedBlock,
					Round:       0,
					Height:      0,
					Signatures:  testutils.RandomSignatures(10),
					Signatories: testutils.RandomSignatories(10),
				},
			}
			Expect(commit.String()).Should(Equal("Commit(Polka(Block=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=,Round=0,Height=0))"))
		})
	})

	Context("when Drop is called on a specific Height", func() {
		It("should remove all SignedPreCommits below the given Height", func() {
			builder := CommitBuilder{}
			for j := 0; j < 10; j++ {
				for i := 0; i < 10; i++ {
					block := Block{Height: 1, Round: Round(i)}
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					signedBlock, err := block.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					precommit := PreCommit{
						Polka: Polka{
							Block:  &signedBlock,
							Height: 1,
							Round:  Round(i),
						},
					}
					signedPreCommit, err := precommit.Sign(signer)
					Expect(err).ShouldNot(HaveOccurred())
					builder.Insert(signedPreCommit)
				}
			}

			block := Block{Height: 2, Round: Round(10)}
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			signedBlock, err := block.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			precommit := PreCommit{
				Polka: Polka{
					Block:  &signedBlock,
					Height: 2,
					Round:  10,
				},
			}
			signedPreCommit, err := precommit.Sign(signer)
			Expect(err).ShouldNot(HaveOccurred())
			builder.Insert(signedPreCommit)

			commit, ok := builder.Commit(1, 10)
			Expect(ok).To(BeTrue())
			Expect(commit.Polka.Round).To(Equal(Round(9)))

			builder.Drop(2)

			commit, ok = builder.Commit(1, 1)
			Expect(ok).To(BeFalse())

			commit, ok = builder.Commit(2, 1)
			Expect(ok).To(BeTrue())
			Expect(commit.Polka.Round).To(Equal(Round(10)))
		})
	})
})
