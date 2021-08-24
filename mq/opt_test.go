package mq_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/mq"

	"go.uber.org/zap"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MQ Opts", func() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	Context("MQ Opts", func() {
		Specify("with default opts", func() {
			opts := mq.DefaultOptions()
			Expect(opts.MaxCapacity).To(Equal(1000))
		})

		Specify("with logger", func() {
			logger := zap.NewExample()
			_ = mq.DefaultOptions().WithLogger(logger)
		})

		Specify("with max capacity", func() {
			loop := func() bool {
				capacity := int(r.Int63())
				opts := mq.DefaultOptions().WithMaxCapacity(capacity)
				Expect(opts.MaxCapacity).To(Equal(capacity))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})
})
