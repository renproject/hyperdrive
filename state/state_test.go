package state_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/renproject/hyperdrive/state"
)

var _ = Describe("States", func() {
	Context("when using WaitingForPropose", func() {
		It("should implement the State interface", func() {
			state.WaitingForPropose{}.IsState()
		})
	})

	Context("when using WaitingForPolka", func() {
		It("should implement the State interface", func() {
			state.WaitingForPolka{}.IsState()
		})
	})

	Context("when using WaitingForCommit", func() {
		It("should implement the State interface", func() {
			state.WaitingForCommit{}.IsState()
		})
	})
})
