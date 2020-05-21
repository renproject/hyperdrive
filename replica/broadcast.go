package replica

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

type signer struct {
	broadcaster process.Broadcaster
	privKey     ecdsa.PrivateKey
}

func newSigner(broadcaster process.Broadcaster, privKey ecdsa.PrivateKey) process.Broadcaster {
	return &signer{
		broadcaster: broadcaster,
		privKey:     privKey,
	}
}

// Broadcast implements the `process.Broadcaster` interface.
func (broadcaster *signer) Broadcast(m process.Message) {
	if err := process.Sign(m, broadcaster.privKey); err != nil {
		panic(fmt.Errorf("invariant violation: error broadcasting message: %v", err))
	}
	broadcaster.broadcaster.Broadcast(m)
}

// Cast implements the `process.Broadcaster` interface.
func (broadcaster *signer) Cast(to id.Signatory, m process.Message) {
	if err := process.Sign(m, broadcaster.privKey); err != nil {
		panic(fmt.Errorf("invariant violation: error broadcasting message: %v", err))
	}
	broadcaster.broadcaster.Cast(to, m)
}
