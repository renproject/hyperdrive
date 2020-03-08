package replica

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"testing/quick"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/id"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/process"
)

type mockBroadcaster struct {
	broadcastMessages chan<- Message
	castMessages      chan<- Message
}

func (m *mockBroadcaster) Broadcast(message Message) {
	m.broadcastMessages <- message
}

func (m *mockBroadcaster) Cast(to id.Signatory, message Message) {
	m.castMessages <- message
}

func newMockBroadcaster() (Broadcaster, chan Message, chan Message) {
	broadcastMessages := make(chan Message, 1)
	castMessages := make(chan Message, 1)
	return &mockBroadcaster{
		broadcastMessages: broadcastMessages,
		castMessages:      castMessages,
	}, broadcastMessages, castMessages
}

var _ = Describe("Broadcaster", func() {
	Context("when broadcasting a message", func() {
		It("should sign the message and then broadcast it", func() {
			test := func(shard Shard) bool {
				key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
				Expect(err).ToNot(HaveOccurred())
				broadcaster, broadcastMessages, _ := newMockBroadcaster()
				signer := newSigner(broadcaster, shard, *key)

				msg := RandomMessage(RandomMessageType())
				signer.Broadcast(msg)

				var message Message
				Eventually(broadcastMessages, 2*time.Second).Should(Receive(&message))
				Expect(bytes.Equal(message.Shard[:], shard[:])).Should(BeTrue())
				Expect(process.Verify(message.Message)).Should(Succeed())

				return true

			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})

	Context("when casting a message", func() {
		It("should sign the message and then cast it", func() {
			test := func(shard Shard) bool {
				key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
				Expect(err).ToNot(HaveOccurred())
				broadcaster, _, castMessages := newMockBroadcaster()
				signer := newSigner(broadcaster, shard, *key)

				msg := RandomMessage(RandomMessageType())
				signer.Cast(id.Signatory{}, msg)

				var message Message
				Eventually(castMessages, 2*time.Second).Should(Receive(&message))
				Expect(bytes.Equal(message.Shard[:], shard[:])).Should(BeTrue())
				Expect(process.Verify(message.Message)).Should(Succeed())

				return true
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})
})
