package replica_test

import (
	"testing/quick"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/replica"
	. "github.com/renproject/hyperdrive/testutil"

	"github.com/renproject/id"
)

var _ = Describe("roundRobinScheduler", func() {
	Context("when asking for proposer of a height and round", func() {
		It("should return a valid signatory in a round-robin way", func() {
			test := func(signatories id.Signatories) {
				scheduler := NewRoundRobinScheduler(signatories)
				sigs := map[id.Signatory]struct{}{}
				for _, sig:= range signatories{
					sigs[sig] = struct{}{}
				}

				// It will always give us a signatory we passed
				for i := 0; i < 100 ;i ++ {
					height, round := RandomHeight(), RandomRound()
					sig := scheduler.Schedule(height, round)
					_, ok := sigs[sig]
					Expect(ok).Should(BeTrue())
				}
			}

			Expect(quick.Check(test,nil)).Should(BeTrue())
		})
	})
})
