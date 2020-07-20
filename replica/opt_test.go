package replica_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/mq"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/timer"

	"go.uber.org/zap"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Replica Opts", func() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	Context("Replica Opts", func() {
		Specify("with default opts", func() {
			opts := replica.DefaultOptions()

			Expect(opts.MessageQueueOpts.MaxCapacity).To(Equal(1000))
			Expect(opts.TimerOpts.Timeout).To(Equal(20 * time.Second))
			Expect(opts.TimerOpts.TimeoutScaling).To(Equal(0.5))
		})

		Specify("with logger", func() {
			logger := zap.NewExample()
			_ = replica.DefaultOptions().WithLogger(logger)
		})

		Specify("with timer opts", func() {
			loop := func() bool {
				timeout := time.Duration(r.Intn(100)) * time.Second
				timeoutScaling := r.Float64()
				timerOpts := timer.DefaultOptions().
					WithTimeout(timeout).
					WithTimeoutScaling(timeoutScaling)

				opts := replica.DefaultOptions().WithTimerOptions(timerOpts)
				Expect(opts.TimerOpts.Timeout).To(Equal(timeout))
				Expect(opts.TimerOpts.TimeoutScaling).To(Equal(timeoutScaling))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		Specify("with message queue opts", func() {
			loop := func() bool {
				capacity := int(r.Int63())
				mqOpts := mq.DefaultOptions().WithMaxCapacity(capacity)

				opts := replica.DefaultOptions().WithMqOptions(mqOpts)
				Expect(opts.MessageQueueOpts.MaxCapacity).To(Equal(capacity))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})
})
