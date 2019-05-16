package state_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/renproject/hyperdrive/state"
)

var _ = Describe("Actions", func() {
	Context("when using Propose", func() {
		It("should implement the Action interface", func() {
			state.Propose{}.IsAction()
		})
	})

	Context("when using PreVote", func() {
		It("should implement the Action interface", func() {
			state.PreVote{}.IsAction()
		})
	})

	Context("when using SignedPreVote", func() {
		It("should implement the Action interface", func() {
			state.SignedPreVote{}.IsAction()
		})
	})

	Context("when using PreCommit", func() {
		It("should implement the Action interface", func() {
			state.PreCommit{}.IsAction()
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
