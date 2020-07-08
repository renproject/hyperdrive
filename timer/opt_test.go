package timer_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/timer"

	"go.uber.org/zap"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Timer Opts", func() {
	rand.Seed(int64(time.Now().Nanosecond()))

	Context("Linear Timer", func() {
		Specify("with default options", func() {
			defaultOpts := timer.DefaultOptions()
			Expect(defaultOpts.Timeout).To(Equal(20 * time.Second))
			Expect(defaultOpts.TimeoutScaling).To(Equal(0.5))
		})

		Specify("with logger", func() {
			logger := zap.NewExample()
			_ = timer.DefaultOptions().WithLogger(logger)
		})

		Specify("with timeout", func() {
			loop := func() bool {
				timeout := time.Duration(rand.Intn(100)) * time.Second
				opts := timer.DefaultOptions().WithTimeout(timeout)
				Expect(opts.Timeout).To(Equal(timeout))
				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		Specify("with timeout", func() {
			loop := func() bool {
				timeoutScaling := rand.Float64()
				opts := timer.DefaultOptions().WithTimeoutScaling(timeoutScaling)
				Expect(opts.TimeoutScaling).To(Equal(timeoutScaling))
				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})
})
