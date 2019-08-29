package replica

import (
	"math/rand"
	"testing/quick"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
)

var _ = Describe("roundRobinScheduler", func() {
	Context("when asking the proposer of given height and round", func() {
		It("should return a valid signatory in a round-robin way", func() {
			test := func(signatories id.Signatories) bool {
				scheduler := newRoundRobinScheduler(signatories)
				sigs := map[id.Signatory]struct{}{}
				for _, sig := range signatories {
					sigs[sig] = struct{}{}
				}

				// It will always give us a signatory we passed
				for i := 0; i < 100; i++ {
					height := block.Height(rand.Int())
					round := block.Round(rand.Int())
					sig := scheduler.Schedule(height, round)
					if len(sigs) == 0 {
						Expect(sig.Equal(block.InvalidSignatory)).Should(BeTrue())
					} else {
						_, ok := sigs[sig]
						Expect(ok).Should(BeTrue())
					}
				}
				return true
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})

	Context("when rebasing", func() {
		It("should return the signatory in the rebase block.", func() {
			test := func(signatories, rebaseSigs id.Signatories) bool {
				scheduler := newRoundRobinScheduler(signatories)

				scheduler.rebase(rebaseSigs)
				sigs := map[id.Signatory]struct{}{}
				for _, sig := range rebaseSigs {
					sigs[sig] = struct{}{}
				}

				// It will always give us a signatory we passed
				for i := 0; i < 100; i++ {
					height := block.Height(rand.Int())
					round := block.Round(rand.Int())
					sig := scheduler.Schedule(height, round)
					if len(sigs) == 0 {
						Expect(sig.Equal(block.InvalidSignatory)).Should(BeTrue())
					} else {
						_, ok := sigs[sig]
						Expect(ok).Should(BeTrue())
					}
				}
				return true
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})
})
