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

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/process"
)

type mockBroadcaster struct {
	messages chan<- Message
}

func (m *mockBroadcaster) Broadcast(message Message) {
	m.messages <- message
}

func newMockBroadcaster() (Broadcaster, chan Message) {
	messages := make(chan Message, 1)
	return &mockBroadcaster{
		messages: messages,
	}, messages
}

var _ = Describe("signer", func() {
	Context("when broadcasting message", func() {
		It("should sign the message and then broadcast it", func() {
			test := func(shard Shard) bool {
				key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
				Expect(err).NotTo(HaveOccurred())
				broadcaster, messages := newMockBroadcaster()
				signer := newSigner(broadcaster, shard, *key)

				msg := RandomMessage(RandomMessageType())
				signer.Broadcast(msg)

				var message Message
				Eventually(messages, 2*time.Second).Should(Receive(&message))
				Expect(bytes.Equal(message.Shard[:], shard[:])).Should(BeTrue())
				Expect(process.Verify(message.Message)).Should(Succeed())
				return true

			}

			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})
})
