package schedule_test

import (
	"testing/quick"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/schedule"
	"github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/id"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Round-robin scheduler", func() {
	Context("when scheduling `n` times", func() {
		Context("when only incrementing the height", func() {
			It("should schedule each signatories `n` times", func() {
				f := func(n int) bool {
					// Clamp n to be positive, and less than 100.
					if n < 0 {
						n = -n
					}
					n = n % 100

					// Setup the scheduler.
					signatories := testutil.RandomSignatories()
					scheduler := schedule.RoundRobin(signatories)

					// Call the Schedule method n times for each signatory,
					// incrementing the Height by exactly 1 each time.
					counts := map[id.Signatory]int{}
					for i := 0; i < n*len(signatories); i++ {
						proposer := scheduler.Schedule(block.Height(i), block.Round(0))
						counts[proposer]++
					}

					// Expect each Signatory to have been returned n times.
					for _, signatory := range signatories {
						Expect(counts[signatory]).To(Equal(n))
					}
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})

		Context("when only incrementing the round", func() {
			It("should schedule each signatories `n` times", func() {
				f := func(n int) bool {
					// Clamp n to be positive, and less than 100.
					if n < 0 {
						n = -n
					}
					n = n % 100

					// Setup the scheduler.
					signatories := testutil.RandomSignatories()
					scheduler := schedule.RoundRobin(signatories)

					// Call the Schedule method n times for each signatory,
					// incrementing the Round by exactly 1 each time.
					counts := map[id.Signatory]int{}
					for i := 0; i < n*len(signatories); i++ {
						proposer := scheduler.Schedule(block.Height(0), block.Round(i))
						counts[proposer]++
					}

					// Expect each Signatory to have been returned n times.
					for _, signatory := range signatories {
						Expect(counts[signatory]).To(Equal(n))
					}
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})

		Context("when the underlying signatories are nil", func() {
			It("should return the `InvalidSignatory`", func() {
				f := func() bool {
					scheduler := schedule.RoundRobin(nil)
					proposer := scheduler.Schedule(testutil.RandomHeight(), testutil.RandomRound())
					Expect(proposer).To(Equal(block.InvalidSignatory))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})

		Context("when the underlying signatories are empty", func() {
			It("should return the `InvalidSignatory`", func() {
				f := func() bool {
					scheduler := schedule.RoundRobin(id.Signatories{})
					proposer := scheduler.Schedule(testutil.RandomHeight(), testutil.RandomRound())
					Expect(proposer).To(Equal(block.InvalidSignatory))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})

		Context("when the height is invalid", func() {
			It("should return the `InvalidSignatory`", func() {
				f := func() bool {
					scheduler := schedule.RoundRobin(testutil.RandomSignatories())
					proposer := scheduler.Schedule(block.InvalidHeight, testutil.RandomRound())
					Expect(proposer).To(Equal(block.InvalidSignatory))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})

		Context("when the round is invalid", func() {
			It("should return the `InvalidSignatory`", func() {
				f := func() bool {
					scheduler := schedule.RoundRobin(testutil.RandomSignatories())
					proposer := scheduler.Schedule(testutil.RandomHeight(), block.InvalidRound)
					Expect(proposer).To(Equal(block.InvalidSignatory))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})
	})

	Context("when rebasing", func() {
		Context("when scheduling after the rebase", func() {
			It("should schedule new signatories, and not schedule old signatories", func() {
				f := func(n int) bool {
					// Clamp n to be positive, and less than 100.
					if n < 0 {
						n = -n
					}
					n = n % 100

					// Setup the scheduler.
					oldSignatories := testutil.RandomSignatories()
					scheduler := schedule.RoundRobin(oldSignatories)

					// Change to new signatories.
					newSignatories := testutil.RandomSignatories()
					scheduler.Rebase(newSignatories)

					// Call the Schedule method n time, incrementing the Round
					// by exactly 1 each time.
					counts := map[id.Signatory]int{}
					for i := 0; i < n*len(newSignatories); i++ {
						proposer := scheduler.Schedule(block.Height(i), block.Round(0))
						counts[proposer]++
					}

					// Expect the signatories to be scheduled.
					for _, signatory := range newSignatories {
						Expect(counts[signatory]).To(Equal(n))
					}
					// Expect none of the old signatories to be scheduled.
					for _, signatory := range oldSignatories {
						Expect(counts[signatory]).To(Equal(0))
					}
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})

		Context("when the new underlying signatories are nil", func() {
			It("should return the `InvalidSignatory`", func() {
				f := func() bool {
					scheduler := schedule.RoundRobin(testutil.RandomSignatories())
					scheduler.Rebase(nil)
					proposer := scheduler.Schedule(testutil.RandomHeight(), testutil.RandomRound())
					Expect(proposer).To(Equal(block.InvalidSignatory))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})

		Context("when the new underlying signatories are empty", func() {
			It("should return the `InvalidSignatory`", func() {
				f := func() bool {
					scheduler := schedule.RoundRobin(testutil.RandomSignatories())
					scheduler.Rebase(id.Signatories{})
					proposer := scheduler.Schedule(testutil.RandomHeight(), testutil.RandomRound())
					Expect(proposer).To(Equal(block.InvalidSignatory))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})
	})
})
