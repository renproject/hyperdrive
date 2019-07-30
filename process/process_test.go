package process_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Process", func() {
	Context("when implementing the test suite needs to be done", func() {
		It("should always fail the test", func() {
			Expect(true).To(BeFalse())
		})
	})
})
