package timer_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/processutil"
	"github.com/renproject/hyperdrive/timer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Timer", func() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	Context("Timer", func() {
		Context("without a timeout scaling factor", func() {
			Specify("on timeout propose", func() {
				loop := func() bool {
					// 5 millisecond <= timeout <= 20 millisecond
					timeout := time.Duration(5+r.Intn(16)) * time.Millisecond
					timeoutScaling := 0.0
					opts := timer.DefaultOptions().
						WithTimeout(timeout).
						WithTimeoutScaling(timeoutScaling)
					onProposeTimeoutChan := make(chan timer.Timeout, 1)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 1)
					onPrecommitTimeoutChan := make(chan timer.Timeout, 1)
					constantTimer := timer.NewLinearTimer(opts, onProposeTimeoutChan, onPrevoteTimeoutChan, onPrecommitTimeoutChan)

					// timeout should be the same for any round/height
					height := processutil.RandomHeight(r)
					round := processutil.RandomRound(r)

					constantTimer.TimeoutPropose(height, round)

					// message will be received at least by that time
					time.Sleep(timeout - (10 * time.Millisecond))
					select {
					case _ = <-onProposeTimeoutChan:
						// the channel is empty, so should not reach here
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					// message will be received at least by that time
					time.Sleep(20 * time.Millisecond)
					select {
					case timeoutFor := <-onProposeTimeoutChan:
						Expect(timeoutFor.Height).To(Equal(height))
						Expect(timeoutFor.Round).To(Equal(round))
					default:
						// this should not happen
						Expect(true).ToNot(BeTrue())
					}

					// no other channel should have received any message
					select {
					case _ = <-onPrevoteTimeoutChan:
						Expect(true).ToNot(BeTrue())
					case _ = <-onPrecommitTimeoutChan:
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})

			Specify("on timeout prevote", func() {
				loop := func() bool {
					// 5 millisecond <= timeout <= 20 millisecond
					timeout := time.Duration(5+r.Intn(16)) * time.Millisecond
					timeoutScaling := 0.0
					opts := timer.DefaultOptions().
						WithTimeout(timeout).
						WithTimeoutScaling(timeoutScaling)
					onProposeTimeoutChan := make(chan timer.Timeout, 1)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 1)
					onPrecommitTimeoutChan := make(chan timer.Timeout, 1)
					constantTimer := timer.NewLinearTimer(opts, onProposeTimeoutChan, onPrevoteTimeoutChan, onPrecommitTimeoutChan)

					// timeout should be the same for any round/height
					height := processutil.RandomHeight(r)
					round := processutil.RandomRound(r)

					constantTimer.TimeoutPrevote(height, round)

					// message will be received at least by that time
					time.Sleep(timeout - (10 * time.Millisecond))
					select {
					case _ = <-onPrevoteTimeoutChan:
						// the channel is empty, so should not reach here
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					// message will be received at least by that time
					time.Sleep(20 * time.Millisecond)
					select {
					case timeoutFor := <-onPrevoteTimeoutChan:
						Expect(timeoutFor.Height).To(Equal(height))
						Expect(timeoutFor.Round).To(Equal(round))
					default:
						// this should not happen
						Expect(true).ToNot(BeTrue())
					}

					// no other channel should have received any message
					select {
					case _ = <-onProposeTimeoutChan:
						Expect(true).ToNot(BeTrue())
					case _ = <-onPrecommitTimeoutChan:
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})

			Specify("on timeout precommit", func() {
				loop := func() bool {
					// 5 millisecond <= timeout <= 20 millisecond
					timeout := time.Duration(5+r.Intn(16)) * time.Millisecond
					timeoutScaling := 0.0
					opts := timer.DefaultOptions().
						WithTimeout(timeout).
						WithTimeoutScaling(timeoutScaling)
					onProposeTimeoutChan := make(chan timer.Timeout, 1)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 1)
					onPrecommitTimeoutChan := make(chan timer.Timeout, 1)
					constantTimer := timer.NewLinearTimer(opts, onProposeTimeoutChan, onPrevoteTimeoutChan, onPrecommitTimeoutChan)

					// timeout should be the same for any round/height
					height := processutil.RandomHeight(r)
					round := processutil.RandomRound(r)

					constantTimer.TimeoutPrecommit(height, round)

					// message will be received at least by that time
					time.Sleep(timeout - (10 * time.Millisecond))
					select {
					case _ = <-onPrecommitTimeoutChan:
						// the channel is empty, so should not reach here
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					// message will be received at least by that time
					time.Sleep(20 * time.Millisecond)
					select {
					case timeoutFor := <-onPrecommitTimeoutChan:
						Expect(timeoutFor.Height).To(Equal(height))
						Expect(timeoutFor.Round).To(Equal(round))
					default:
						// this should not happen
						Expect(true).ToNot(BeTrue())
					}

					// no other channel should have received any message
					select {
					case _ = <-onProposeTimeoutChan:
						Expect(true).ToNot(BeTrue())
					case _ = <-onPrevoteTimeoutChan:
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})

		Context("with a timeout scaling factor", func() {
			Specify("on timeout propose", func() {
				loop := func() bool {
					timeout := 5 * time.Millisecond
					timeoutScaling := r.Float64() / 2.0
					opts := timer.DefaultOptions().
						WithTimeout(timeout).
						WithTimeoutScaling(timeoutScaling)
					onProposeTimeoutChan := make(chan timer.Timeout, 1)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 1)
					onPrecommitTimeoutChan := make(chan timer.Timeout, 1)
					constantTimer := timer.NewLinearTimer(opts, onProposeTimeoutChan, onPrevoteTimeoutChan, onPrecommitTimeoutChan)

					// timeout should scale up linearly with round
					height := processutil.RandomHeight(r)
					round := process.Round(r.Intn(20))

					expectedTimeout := timeout + (timeout * (time.Duration((float64(round) * timeoutScaling))))

					constantTimer.TimeoutPropose(height, round)

					// message will not be received by that time
					time.Sleep(expectedTimeout - (10 * time.Millisecond))
					select {
					case _ = <-onProposeTimeoutChan:
						// the channel is empty, so should not reach here
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					// message will be received at least by that time
					time.Sleep(20 * time.Millisecond)
					select {
					case timeoutFor := <-onProposeTimeoutChan:
						Expect(timeoutFor.Height).To(Equal(height))
						Expect(timeoutFor.Round).To(Equal(round))
					default:
						// this should not happen
						Expect(true).ToNot(BeTrue())
					}

					// no other channel should have received any message
					select {
					case _ = <-onPrevoteTimeoutChan:
						Expect(true).ToNot(BeTrue())
					case _ = <-onPrecommitTimeoutChan:
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})

			Specify("on timeout prevote", func() {
				loop := func() bool {
					timeout := 5 * time.Millisecond
					timeoutScaling := r.Float64() / 2.0
					opts := timer.DefaultOptions().
						WithTimeout(timeout).
						WithTimeoutScaling(timeoutScaling)
					onProposeTimeoutChan := make(chan timer.Timeout, 1)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 1)
					onPrecommitTimeoutChan := make(chan timer.Timeout, 1)
					constantTimer := timer.NewLinearTimer(opts, onProposeTimeoutChan, onPrevoteTimeoutChan, onPrecommitTimeoutChan)

					// timeout should scale up linearly with round
					height := processutil.RandomHeight(r)
					round := process.Round(r.Intn(20))

					expectedTimeout := timeout + (timeout * (time.Duration((float64(round) * timeoutScaling))))

					constantTimer.TimeoutPrevote(height, round)

					// message will not be received by that time
					time.Sleep(expectedTimeout - (10 * time.Millisecond))
					select {
					case _ = <-onPrevoteTimeoutChan:
						// the channel is empty, so should not reach here
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					// message will be received at least by that time
					time.Sleep(20 * time.Millisecond)
					select {
					case timeoutFor := <-onPrevoteTimeoutChan:
						Expect(timeoutFor.Height).To(Equal(height))
						Expect(timeoutFor.Round).To(Equal(round))
					default:
						// this should not happen
						Expect(true).ToNot(BeTrue())
					}

					// no other channel should have received any message
					select {
					case _ = <-onProposeTimeoutChan:
						Expect(true).ToNot(BeTrue())
					case _ = <-onPrecommitTimeoutChan:
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})

			Specify("on timeout precommit", func() {
				loop := func() bool {
					timeout := 5 * time.Millisecond
					timeoutScaling := r.Float64() / 2.0
					opts := timer.DefaultOptions().
						WithTimeout(timeout).
						WithTimeoutScaling(timeoutScaling)
					onProposeTimeoutChan := make(chan timer.Timeout, 1)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 1)
					onPrecommitTimeoutChan := make(chan timer.Timeout, 1)
					constantTimer := timer.NewLinearTimer(opts, onProposeTimeoutChan, onPrevoteTimeoutChan, onPrecommitTimeoutChan)

					// timeout should scale up linearly with round
					height := processutil.RandomHeight(r)
					round := process.Round(r.Intn(20))

					expectedTimeout := timeout + (timeout * (time.Duration((float64(round) * timeoutScaling))))

					constantTimer.TimeoutPrecommit(height, round)

					// message will not be received by that time
					time.Sleep(expectedTimeout - (10 * time.Millisecond))
					select {
					case _ = <-onPrecommitTimeoutChan:
						// the channel is empty, so should not reach here
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					// message will be received at least by that time
					time.Sleep(20 * time.Millisecond)
					select {
					case timeoutFor := <-onPrecommitTimeoutChan:
						Expect(timeoutFor.Height).To(Equal(height))
						Expect(timeoutFor.Round).To(Equal(round))
					default:
						// this should not happen
						Expect(true).ToNot(BeTrue())
					}

					// no other channel should have received any message
					select {
					case _ = <-onPrevoteTimeoutChan:
						Expect(true).ToNot(BeTrue())
					case _ = <-onProposeTimeoutChan:
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})
	})
})
