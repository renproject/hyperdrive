package state_test

import (
	"github.com/renproject/hyperdrive/state"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Actions", func() {
	Context("when using Propose", func() {
		It("should implement the Action interface", func() {
			state.Propose{}.IsAction()
		})
	})

	Context("when using SignedPreVote", func() {
		It("should implement the Action interface", func() {
			state.SignedPreVote{}.IsAction()
		})
	})

	Context("when using SignedPreCommit", func() {
		It("should implement the Action interface", func() {
			state.SignedPreCommit{}.IsAction()
		})
	})

	Context("when using Commit", func() {
		It("should implement the Action interface", func() {
			state.Commit{}.IsAction()
		})
	})
})
