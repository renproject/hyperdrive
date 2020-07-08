package scheduler_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/scheduler"
	"github.com/renproject/id"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scheduler", func() {
	rand.Seed(int64(time.Now().Nanosecond()))

	Context("when scheduling", func() {
		var first, second, third id.Signatory
		var roundRobinScheduler process.Scheduler

		BeforeEach(func() {
			first = id.NewPrivKey().Signatory()
			second = id.NewPrivKey().Signatory()
			third = id.NewPrivKey().Signatory()
			roundRobinScheduler = scheduler.NewRoundRobin(
				[]id.Signatory{
					first,
					second,
					third,
				},
			)
		})

		It("should panic for an invalid height", func() {
			loop := func() bool {
				invalidHeight := process.Height(-rand.Int63())
				round := process.Round(rand.Int63())
				Expect(func() {
					roundRobinScheduler.Schedule(invalidHeight, round)
				}).To(PanicWith("invalid height"))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should panic for an invalid round", func() {
			loop := func() bool {
				height := process.Height(rand.Int63())
				invalidRound := process.Round(-rand.Int63())
				Expect(func() {
					roundRobinScheduler.Schedule(height, invalidRound)
				}).To(PanicWith("invalid round"))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())

			height := process.Height(rand.Int63())
			Expect(func() {
				roundRobinScheduler.Schedule(height, process.InvalidRound)
			}).To(PanicWith("invalid round"))
		})

		It("should panic for no signatories", func() {
			loop := func() bool {
				roundRobinScheduler = scheduler.NewRoundRobin([]id.Signatory{})
				height := process.Height(rand.Int63())
				round := process.Round(rand.Int63())
				Expect(func() {
					roundRobinScheduler.Schedule(height, round)
				}).To(PanicWith("no processes to schedule"))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should schedule correctly for a single signatory", func() {
			loop := func() bool {
				onlyOne := id.NewPrivKey().Signatory()
				roundRobinScheduler = scheduler.NewRoundRobin(
					[]id.Signatory{onlyOne},
				)

				for t := 0; t <= 20; t++ {
					height := process.Height(rand.Int63())
					round := process.Round(rand.Int63())
					Expect(roundRobinScheduler.Schedule(height, round)).To(Equal(onlyOne))
				}

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should schedule correctly for more than one signatories", func() {
			loop := func() bool {
				// create a list of signatories
				n := 2 + rand.Intn(12)
				signatories := make([]id.Signatory, n)
				for i := 0; i < n; i++ {
					signatories[i] = id.NewPrivKey().Signatory()
				}

				// instantiate the round robin scheduler
				roundRobinScheduler = scheduler.NewRoundRobin(signatories)

				// increment height while round is the same
				maxHeight := 13 + rand.Intn(28)
				for i := 1; i <= maxHeight; i++ {
					Expect(roundRobinScheduler.Schedule(process.Height(i), process.Round(0))).
						To(Equal(signatories[i%n]))
				}

				// increment round while height is the same
				maxRound := 13 + rand.Intn(28)
				for i := 0; i < maxRound; i++ {
					Expect(roundRobinScheduler.Schedule(process.Height(1), process.Round(i))).
						To(Equal(signatories[(i+1)%n]))
				}

				// increment both height and round
				for i := 2; i <= maxHeight; i++ {
					for j := 1; j < maxRound; j++ {
						Expect(roundRobinScheduler.Schedule(process.Height(i), process.Round(j))).
							To(Equal(signatories[(i+j)%n]))
					}
				}

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})
})
