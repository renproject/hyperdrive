package timer_test

import (
	"bytes"
	"math/rand"
	"strconv"
	"strings"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/timer"

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

		// Specify("with log level", func() {
		// 	loop := func() bool {
		// 		l := rand.Intn(7)
		// 		logLevel := logrus.Level(l)
		// 		opts := timer.DefaultOptions().WithLogLevel(logLevel)
		//
		// 		logger := opts.Logger.(*logrus.Entry)
		// 		Expect(logger.Level).To(Equal(logLevel))
		//
		// 		return true
		// 	}
		// 	Expect(quick.Check(loop, nil)).To(Succeed())
		// })

		Specify("with log output", func() {
			loop := func() bool {
				buf := bytes.NewBuffer([]byte{})
				opts := timer.DefaultOptions().WithLogOutput(buf)

				r := rand.Int()
				opts.Logger.Printf("%d", r)
				Expect(strings.Contains(buf.String(), strconv.Itoa(r))).To(BeTrue())
				Expect(func() {
					opts.Logger.Panicln("some reason")
				}).To(Panic())

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
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
