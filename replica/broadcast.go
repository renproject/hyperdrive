package replica

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/renproject/hyperdrive/process"
)

// A Broadcaster is used to send signed, shard-specific, Messages to all
// Replicas in the network.
type Broadcaster interface {
	Broadcast(Message)
}

type signer struct {
	broadcaster Broadcaster
	shard       Shard
	privKey     ecdsa.PrivateKey
}

// newSigner returns a `process.Broadcaster` that accepts `process.Messages`,
// signs them, sssociated them with a Shard, and re-broadcasts them.
func newSigner(broadcaster Broadcaster, shard Shard, privKey ecdsa.PrivateKey) process.Broadcaster {
	return &signer{
		broadcaster: broadcaster,
		shard:       shard,
		privKey:     privKey,
	}
}

// Broadcast implements the `process.Broadcaster` interface.
func (broadcaster *signer) Broadcast(m process.Message) {
	if err := process.Sign(m, broadcaster.privKey); err != nil {
		panic(fmt.Errorf("invariant violation: error broadcasting message: %v", err))
	}
	broadcaster.broadcaster.Broadcast(Message{
		Message: m,
		Shard:   broadcaster.shard,
	})
}
