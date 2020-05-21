package replica

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing/quick"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/testutil"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

type mockBroadcaster struct {
	broadcastMessages chan<- process.Message
	castMessages      chan<- process.Message
}

func (m *mockBroadcaster) Broadcast(message process.Message) {
	m.broadcastMessages <- message
}

func (m *mockBroadcaster) Cast(to id.Signatory, message process.Message) {
	m.castMessages <- message
}

func newMockBroadcaster() (process.Broadcaster, chan process.Message, chan process.Message) {
	broadcastMessages := make(chan process.Message, 1)
	castMessages := make(chan process.Message, 1)
	return &mockBroadcaster{
		broadcastMessages: broadcastMessages,
		castMessages:      castMessages,
	}, broadcastMessages, castMessages
}

var _ = Describe("Broadcaster", func() {
	Context("when broadcasting a message", func() {
		It("should sign the message and then broadcast it", func() {
			test := func() bool {
				key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
				Expect(err).ToNot(HaveOccurred())
				broadcaster, broadcastMessages, _ := newMockBroadcaster()
				signer := newSigner(broadcaster, *key)

				msg := RandomMessage(RandomMessageType(true))
				signer.Broadcast(msg)

				var message process.Message
				Eventually(broadcastMessages, 2*time.Second).Should(Receive(&message))
				Expect(process.Verify(message)).Should(Succeed())

				return true

			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})

	Context("when casting a message", func() {
		It("should sign the message and then cast it", func() {
			test := func() bool {
				key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
				Expect(err).ToNot(HaveOccurred())
				broadcaster, _, castMessages := newMockBroadcaster()
				signer := newSigner(broadcaster, *key)

				msg := RandomMessage(RandomMessageType(true))
				signer.Cast(id.Signatory{}, msg)

				var message process.Message
				Eventually(castMessages, 2*time.Second).Should(Receive(&message))
				Expect(process.Verify(message)).Should(Succeed())

				return true
			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})
})
