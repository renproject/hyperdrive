package process_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("States", func() {
	Context("when marshaling", func() {
		Context("when marshaling a random state", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})
	})

	Context("when resetting equal states", func() {
		It("should return states that are equal", func() {
			Expect(true).To(BeFalse())
		})
	})

	Context("when resetting unequal states", func() {
		It("should return states that are equal", func() {
			Expect(true).To(BeFalse())
		})
	})

	Context("when resetting the default states", func() {
		It("should return a state that is equal to the default state", func() {
			Expect(true).To(BeFalse())
		})
	})
})
