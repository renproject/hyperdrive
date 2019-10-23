package replica_test

import (
	"encoding/json"

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
					Message: RandomMessage(RandomMessageType()),
					Shard:   Shard{},
				}
				messageBytes, err := message.MarshalBinary()
				Expect(err).ToNot(HaveOccurred())

				var newMessage Message
				Expect(newMessage.UnmarshalBinary(messageBytes)).To(Succeed())
				Expect(newMessage).To(Equal(message))
			}
		})

		It("should equal itself after json marshaling/unmarshaling", func() {
			for i := 0; i < 10; i++ {
				message := Message{
					Message: RandomMessage(RandomMessageType()),
					Shard:   Shard{},
				}
				messageBytes, err := json.Marshal(message)
				Expect(err).ToNot(HaveOccurred())

				var newMessage Message
				Expect(json.Unmarshal(messageBytes, &newMessage)).To(Succeed())
				Expect(newMessage).To(Equal(message))
			}
		})
	})
})
