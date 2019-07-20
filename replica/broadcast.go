package replica

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/renproject/hyperdrive/message"
	"github.com/renproject/hyperdrive/process"
)

func NewSignerBroadcaster(broadcaster process.Broadcaster, privKey ecdsa.PrivateKey) process.Broadcaster {
	return &signerBroadcaster{
		broadcaster: broadcaster,
		privKey:     privKey,
	}
}

type signerBroadcaster struct {
	broadcaster process.Broadcaster
	privKey     ecdsa.PrivateKey
}

func (broadcaster *signerBroadcaster) Broadcast(m message.Message) {
	if err := message.Sign(m, broadcaster.privKey); err != nil {
		panic(fmt.Errorf("invariant violation: error broadcasting message: %v", err))
	}
	broadcaster.broadcaster.Broadcast(m)
}
