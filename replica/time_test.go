package replica

import (
	"math/rand"
	"testing/quick"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
)

func randomStep() process.Step {
	return process.Step(rand.Intn(3) + 1)
}

var _ = Describe("timer", func() {
	Context("when using a backOffTimer", func() {
		It("should return a timeout of certain in a backoff manner", func() {
			test := func() bool {
				max := time.Duration(rand.Int())
				base := time.Duration(rand.Intn(int(max)))
				exp := rand.Float64() + 1
				timer := newBackOffTimer(exp, base, max)

				Expect(timer.Timeout(randomStep(), 0)).Should(Equal(base))

				preTimeout := time.Duration(0)
				for round := block.Round(1); round < 20; round++ {
					timeout := timer.Timeout(randomStep(), round)
					Expect(timeout).Should(BeNumerically("<=", max))
					Expect(timeout).Should(BeNumerically(">=", preTimeout))
					preTimeout = timeout
				}
				return true
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})
})
