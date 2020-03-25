package replica_test

import (
	"github.com/renproject/surge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/replica"
	. "github.com/renproject/hyperdrive/testutil"
)

var _ = Describe("Marshaling", func() {
	Context("when marshaling and unmarshaling a message", func() {
		It("should equal itself after binary marshaling/unmarshaling", func() {
			for i := 0; i < 10; i++ {
				message := Message{
					Message: RandomMessage(RandomMessageType(true)),
					Shard:   Shard{},
				}
				messageBytes, err := surge.ToBinary(message)
				Expect(err).ToNot(HaveOccurred())

				var newMessage Message
				Expect(surge.FromBinary(messageBytes, &newMessage)).To(Succeed())
				Expect(newMessage).To(Equal(message))
			}
		})
	})
})
