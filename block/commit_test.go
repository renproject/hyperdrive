package block_test

import (
	"crypto/rand"
	mathRand "math/rand"

	"github.com/renproject/hyperdrive/sig"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("CommitBuilder", func() {
	Context("when less than the threshold of PreCommits is inserted", func() {
		Context("when PreCommits are inserted at the same height and the same round", func() {
			It("should never return a Commit", func() {
				builder := CommitBuilder{}
				for i := 0; i < 10; i++ {
					builder.Insert(SignedPreCommit{
						PreCommit: PreCommit{
							Polka: Polka{
								Block: &Block{
									Height: 0,
									Round:  0,
								},
								Height: 0,
								Round:  0,
							},
						},
						Signatory: randomSignatory(),
					})
				}
				_, ok := builder.Commit(0, 11)
				Expect(ok).To(BeFalse())
			})
		})

		Context("when PreCommits are inserted at the same height and multiple rounds", func() {
			It("should never return a Commit", func() {
				builder := CommitBuilder{}
				for i := 0; i < 10; i++ {
					builder.Insert(SignedPreCommit{
						PreCommit: PreCommit{
							Polka: Polka{
								Block: &Block{
									Height: 0,
									Round:  Round(i),
								},
								Height: 0,
								Round:  Round(i),
							},
						},
						Signatory: randomSignatory(),
					})
				}
				_, ok := builder.Commit(0, 9)
				Expect(ok).To(BeFalse())
			})
		})

		Context("when PreCommits are inserted at multiple heights and the same round", func() {
			It("should never return a Commit", func() {
				builder := CommitBuilder{}
				for i := 0; i < 10; i++ {
					builder.Insert(SignedPreCommit{
						PreCommit: PreCommit{
							Polka: Polka{
								Block: &Block{
									Height: Height(i),
									Round:  0,
								},
								Height: Height(i),
								Round:  0,
							},
						},
						Signatory: randomSignatory(),
					})
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
					builder.Insert(SignedPreCommit{
						PreCommit: PreCommit{
							Polka: Polka{
								Block: &Block{
									Height: height,
									Round:  round,
								},
								Height: height,
								Round:  round,
							},
						},
						Signatory: randomSignatory(),
					})
				}
				_, ok := builder.Commit(height, 9)
				Expect(ok).To(BeFalse())
			})
		})
	})

	Context("when the threshold of PreCommits is inserted at the same round", func() {
		It("should always return a Commit", func() {
			builder := CommitBuilder{}
			height := Height(mathRand.Intn(10))
			round := Round(mathRand.Intn(100))

			for i := 0; i < 10; i++ {
				builder.Insert(SignedPreCommit{
					PreCommit: PreCommit{
						Polka: Polka{
							Block: &Block{
								Height: height,
								Round:  round,
							},
							Height: height,
							Round:  round,
						},
					},
					Signatory: randomSignatory(),
				})
			}
			_, ok := builder.Commit(height, 9)
			Expect(ok).To(BeTrue())
		})
	})

	Context("when the threshold of PreCommits is inserted at multiple rounds", func() {
		It("should always return a Commit at the latest round", func() {
			builder := CommitBuilder{}
			for j := 0; j < 10; j++ {
				for i := 0; i < 10; i++ {
					builder.Insert(SignedPreCommit{
						PreCommit: PreCommit{
							Polka: Polka{
								Block: &Block{
									Height: 1,
									Round:  Round(i),
								},
								Height: 1,
								Round:  Round(i),
							},
						},
						Signatory: randomSignatory(),
					})
				}
			}

			commit, ok := builder.Commit(1, 10)
			Expect(ok).To(BeTrue())
			Expect(commit.Polka.Round).To(Equal(Round(9)))
		})
	})
})

func randomSignatory() sig.Signatory {
	key := make([]byte, 20)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	signatory := sig.Signatory{}
	copy(signatory[:], key[:])

	return signatory
}

func randomSignatories(n int) []sig.Signatory {
	signatories := []sig.Signatory{}
	for i := 0; i < n; i++ {
		signatories = append(signatories, randomSignatory())
	}
	return signatories
}

func randomSignatures(n int) []sig.Signature {
	signatures := []sig.Signature{}
	for i := 0; i < n; i++ {
		signatures = append(signatures, randomSignature())
	}
	return signatures
}

func randomSignature() sig.Signature {
	key := make([]byte, 65)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	signature := sig.Signature{}
	copy(signature[:], key[:])

	return signature
}
