package block_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("CommitBuilder", func() {
	Context("when less than the threshold of PreCommits is inserted", func() {
		Context("when PreCommits are inserted at the same height and the same round", func() {
			It("should never return a Commit", func() {
				// TODO: Write tests.
				Expect(false)
			})
		})

		Context("when PreCommits are inserted at the same height and multiple rounds", func() {
			It("should never return a Commit", func() {
				// TODO: Write tests.
				Expect(false)
			})
		})

		Context("when PreCommits are inserted at multiple heights and the same round", func() {
			It("should never return a Commit", func() {
				// TODO: Write tests.
				Expect(false)
			})
		})

		Context("when PreCommits are inserted at multiple heights and multiple rounds", func() {
			It("should never return a Commit", func() {
				// TODO: Write tests.
				Expect(false)
			})
		})
	})

	Context("when the threshold of PreCommits is inserted at the same round", func() {
		It("should always return a Commit", func() {
			// TODO: Write tests.
			Expect(false)
		})
	})

	Context("when the threshold of PreCommits is inserted at multiple rounds", func() {
		It("should always return a Commit at the latest round", func() {
			// TODO: Write tests.
			Expect(false)
		})
	})
})
