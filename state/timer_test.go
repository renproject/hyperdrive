package state

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Timer", func() {
	Context("when a new timer is created", func() {
		It("should function as expected", func() {
			numTicksToExpiry := 10
			timer := NewTimer(numTicksToExpiry)
			timer.Activate()
			for i := 0; i < numTicksToExpiry-1; i++ {
				Expect(timer.Tick()).To(BeFalse())
			}
			Expect(timer.Tick()).To(BeTrue())
			timer.Deactivate()
			Expect(timer.IsActive()).To(BeFalse())
		})
	})
})
