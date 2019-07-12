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
				ticks, timedout := timer.Tick()
				Expect(timedout).To(BeFalse())
				Expect(ticks).To(Equal(i + 1))
			}
			_, timedout := timer.Tick()
			Expect(timedout).To(BeTrue())
			timer.Deactivate()
			Expect(timer.IsActive()).To(BeFalse())
		})
	})
})
