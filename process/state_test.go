package process_test

import (
	"encoding/json"
	"testing/quick"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/process"
	. "github.com/renproject/hyperdrive/testutil"

	"github.com/renproject/hyperdrive/block"
)

var _ = Describe("States", func() {
	Context("when marshaling", func() {
		Context("when marshaling a random state", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				test := func() bool {
					state := RandomState()
					data, err := json.Marshal(state)
					Expect(err).NotTo(HaveOccurred())

					var newState State
					Expect(json.Unmarshal(data, &newState)).Should(Succeed())
					return state.Equal(newState) && newState.Equal(state)
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("when resetting a state", func() {
		It("should only reset the locked block, locked round, valid block and valid round", func() {
			test := func() bool {
				state := RandomState()
				heightBeforeReset := state.CurrentHeight
				roundBeforeReset := state.CurrentRound
				stepBeforeReset := state.CurrentStep

				state.Reset()

				Expect(state.CurrentHeight).Should(Equal(heightBeforeReset))
				Expect(state.CurrentRound).Should(Equal(roundBeforeReset))
				Expect(state.CurrentStep).Should(Equal(stepBeforeReset))
				Expect(state.LockedBlock).Should(Equal(block.InvalidBlock))
				Expect(state.LockedRound).Should(Equal(block.InvalidRound))
				Expect(state.ValidBlock).Should(Equal(block.InvalidBlock))
				Expect(state.ValidRound).Should(Equal(block.InvalidRound))

				return true
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})

	Context("when resetting the default states", func() {
		It("should return a state that is equal to the default ", func() {
			test := func() bool {
				state := DefaultState(10)
				state.Reset()

				return state.Equal(DefaultState(10))
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})
})
