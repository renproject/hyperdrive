package sig

import (
	"bytes"
	"crypto/ecdsa"
)

// Hash is the result of SHA3_256
type Hash [32]byte

func (hash Hash) Equal(other Hash) bool {
	return bytes.Equal(hash[:], other[:])
}

type Signature []byte

type Signatures []Signature

type Signatory [40]byte

type Signatories []Signatory

// Signer signs the provided hash, returning a signature
type Signer interface {
	// TODO: Signatory is simply passed through this interface, maybe
	// belongs somewhere else
	Sign(hash Hash) (Signature, Signatory, error)
	PubKey() *ecdsa.PublicKey
}

// Verifier take a hash of the data you want to check and the
// signature, then returns the PublicKey iff it is a signature of the
// provided hash
type Verifier interface {
	Verify(hash Hash, sig Signature) (*ecdsa.PublicKey, error)
}

// Authenticator combines a Signer and Verifier
type Authenticator interface {
	Signer
	Verifier
}
