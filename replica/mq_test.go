package replica_test

import (
	"math/rand"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Message queue", func() {
	Context("when initialising", func() {
		Context("when capacity is less than or equal to zero", func() {
			It("should panic", func() {
				Expect(func() { replica.NewMessageQueue(-1) }).To(Panic())
				Expect(func() { replica.NewMessageQueue(0) }).To(Panic())
			})
		})
	})

	Context("when pushing messages", func() {
		Context("when peeking", func() {
			It("should return the lowest height", func() {
				mq := replica.NewMessageQueue(100)
				minHeight := block.Height(1<<63 - 1)
				for i := 0; i < 100; i++ {
					m := testutil.RandomMessage(testutil.RandomMessageType(true))
					if m.Height() < minHeight {
						minHeight = m.Height()
					}
					mq.Push(m)
				}
				Expect(mq.Peek()).To(Equal(minHeight))
			})
		})

		It("should pop messages in height order", func() {
			mq := replica.NewMessageQueue(100)
			for i := 0; i < 100; i++ {
				m := testutil.RandomMessageWithHeightAndRound(block.Height(i+1), block.Round(rand.Int63()), testutil.RandomMessageType(true))
				mq.Push(m)
			}
			messages := mq.PopUntil(50)
			Expect(messages).To(HaveLen(50))
			Expect(messages[0].Height()).To(Equal(block.Height(1)))
			Expect(messages[len(messages)-1].Height()).To(Equal(block.Height(50)))

			for i := range messages {
				if i < len(messages)-1 {
					Expect(messages[i].Height() < messages[i+1].Height()).To(BeTrue())
				}
			}
		})

		It("should not insert the same message more than once", func() {
			mq := replica.NewMessageQueue(100)
			for i := 0; i < 100; i++ {
				m := testutil.RandomMessageWithHeightAndRound(block.Height(i+1), block.Round(rand.Int63()), testutil.RandomMessageType(true))
				mq.Push(m)
				mq.Push(m)
				mq.Push(m)
			}
			Expect(mq.Len()).To(Equal(100))
		})

		Context("when capacity is full", func() {
			Context("when new messages are higher", func() {
				It("should drop the biggest heights", func() {
					mq := replica.NewMessageQueue(100)
					// Insert too many messages.
					for i := 0; i < 200; i++ {
						m := testutil.RandomMessageWithHeightAndRound(block.Height(i+1), block.Round(rand.Int63()), testutil.RandomMessageType(true))
						mq.Push(m)
					}
					// Expect the length to be equal to the initial capacity.
					Expect(mq.Len()).To(Equal(100))

					// Pop the first 50 messages and expect them to have the first
					// 50 heights, indicating that the highest 100 messages were
					// dropped.
					messages := mq.PopUntil(50)
					Expect(messages).To(HaveLen(50))
					Expect(messages[0].Height()).To(Equal(block.Height(1)))
					Expect(messages[len(messages)-1].Height()).To(Equal(block.Height(50)))

					for i := range messages {
						if i < len(messages)-1 {
							Expect(messages[i].Height() < messages[i+1].Height()).To(BeTrue())
						}
					}
				})
			})

			Context("when new messages are lower", func() {
				It("should drop the biggest heights", func() {
					mq := replica.NewMessageQueue(100)
					// Insert too many messages.
					for i := 200; i >= 0; i-- {
						m := testutil.RandomMessageWithHeightAndRound(block.Height(i+1), block.Round(rand.Int63()), testutil.RandomMessageType(true))
						mq.Push(m)
					}
					// Expect the length to be equal to the initial capacity.
					Expect(mq.Len()).To(Equal(100))

					// Pop the first 50 messages and expect them to have the first
					// 50 heights, indicating that the highest 100 messages were
					// dropped.
					messages := mq.PopUntil(50)
					Expect(messages).To(HaveLen(50))
					Expect(messages[0].Height()).To(Equal(block.Height(1)))
					Expect(messages[len(messages)-1].Height()).To(Equal(block.Height(50)))

					for i := range messages {
						if i < len(messages)-1 {
							Expect(messages[i].Height() < messages[i+1].Height()).To(BeTrue())
						}
					}
				})
			})
		})
	})

	Context("when peeking", func() {
		Context("when the queue is empty", func() {
			It("should return an invalid height", func() {
				mq := replica.NewMessageQueue(100)
				Expect(mq.Peek()).To(Equal(block.InvalidHeight))
			})
		})
	})
})
