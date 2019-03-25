package block_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("PolkaBuilder", func() {
	Context("when less than the threshold of PreVotes is inserted", func() {
		Context("when PreVotes are inserted at the same height and the same round", func() {
			It("should never return a Polka", func() {
				// TODO: Write tests.
				Expect(false)
			})
		})

		Context("when PreVotes are inserted at the same height and multiple rounds", func() {
			It("should never return a Polka", func() {
				// TODO: Write tests.
				Expect(false)
			})
		})

		Context("when PreVotes are inserted at multiple heights and the same round", func() {
			It("should never return a Polka", func() {
				// TODO: Write tests.
				Expect(false)
			})
		})

		Context("when PreVotes are inserted at multiple heights and multiple rounds", func() {
			It("should never return a Polka", func() {
				// TODO: Write tests.
				Expect(false)
			})
		})
	})

	Context("when the threshold of PreVotes is inserted at the same round", func() {
		It("should always return a Polka", func() {
			// TODO: Write tests.
			Expect(false)
		})
	})

	Context("when the threshold of PreVotes is inserted at multiple rounds", func() {
		It("should always return a Polka at the latest round", func() {
			// TODO: Write tests.
			Expect(false)
		})
	})
})
