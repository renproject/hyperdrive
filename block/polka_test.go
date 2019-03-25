package block_test

import (
	mathRand "math/rand"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("PolkaBuilder", func() {
	Context("when less than the threshold of PreVotes is inserted", func() {
		Context("when PreVotes are inserted at the same height and the same round", func() {
			It("should never return a Polka", func() {
				builder := PolkaBuilder{}
				for i := 0; i < 10; i++ {
					builder.Insert(SignedPreVote{
						PreVote: PreVote{
							Block: &Block{
								Height: 0,
								Round:  0,
							},
							Height: 0,
							Round:  0,
						},
						Signatory: randomSignatory(),
					})
				}
				_, ok := builder.Polka(0, 11)
				Expect(ok).To(BeFalse())
			})
		})

		Context("when PreVotes are inserted at the same height and multiple rounds", func() {
			It("should never return a Polka", func() {
				builder := PolkaBuilder{}
				for i := 0; i < 10; i++ {
					builder.Insert(SignedPreVote{
						PreVote: PreVote{
							Block: &Block{
								Height: 0,
								Round:  Round(i),
							},
							Height: 0,
							Round:  Round(i),
						},
						Signatory: randomSignatory(),
					})
				}
				_, ok := builder.Polka(0, 9)
				Expect(ok).To(BeFalse())
			})
		})

		Context("when PreVotes are inserted at multiple heights and the same round", func() {
			It("should never return a Polka", func() {
				builder := PolkaBuilder{}
				for i := 0; i < 10; i++ {
					builder.Insert(SignedPreVote{
						PreVote: PreVote{
							Block: &Block{
								Height: Height(i),
								Round:  0,
							},
							Height: Height(i),
							Round:  0,
						},
						Signatory: randomSignatory(),
					})
				}
				_, ok := builder.Polka(0, 9)
				Expect(ok).To(BeFalse())
			})
		})

		Context("when PreVotes are inserted at multiple heights and multiple rounds", func() {
			It("should never return a Polka", func() {
				builder := PolkaBuilder{}
				height := Height(mathRand.Intn(100))
				for i := 0; i < 10; i++ {
					height = Height(mathRand.Intn(100))
					round := Round(mathRand.Intn(100))
					builder.Insert(SignedPreVote{
						PreVote: PreVote{
							Block: &Block{
								Height: height,
								Round:  round,
							},
							Height: height,
							Round:  round,
						},
						Signatory: randomSignatory(),
					})
				}
				_, ok := builder.Polka(height, 9)
				Expect(ok).To(BeFalse())
			})
		})
	})

	Context("when the threshold of PreVotes is inserted at the same round", func() {
		It("should always return a Polka", func() {
			builder := PolkaBuilder{}
			height := Height(mathRand.Intn(10))
			round := Round(mathRand.Intn(100))

			for i := 0; i < 10; i++ {
				builder.Insert(SignedPreVote{
					PreVote: PreVote{
						Block: &Block{
							Height: height,
							Round:  round,
						},
						Height: height,
						Round:  round,
					},
					Signatory: randomSignatory(),
				})
			}
			_, ok := builder.Polka(height, 9)
			Expect(ok).To(BeTrue())
		})
	})

	Context("when the threshold of PreVotes is inserted at multiple rounds", func() {
		It("should always return a Polka at the latest round", func() {
			builder := PolkaBuilder{}
			for j := 0; j < 10; j++ {
				for i := 0; i < 10; i++ {
					builder.Insert(SignedPreVote{
						PreVote: PreVote{
							Block: &Block{
								Height: 1,
								Round:  Round(i),
							},
							Height: 1,
							Round:  Round(i),
						},
						Signatory: randomSignatory(),
					})
				}
			}

			polka, ok := builder.Polka(1, 10)
			Expect(ok).To(BeTrue())
			Expect(polka.Round).To(Equal(Round(9)))
		})
	})
})
