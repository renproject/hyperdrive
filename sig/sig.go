package sig

import (
	"bytes"
)

// Hash is the result of Keccak256
type Hash [32]byte

// Equal compares two `Hash`
func (hash Hash) Equal(other Hash) bool {
	return bytes.Equal(hash[:], other[:])
}

// Signature produced by `Sign`
type Signature [65]byte

// Signatures is an array of Signature
type Signatures []Signature

// Signatory is the last 20 bytes of a Public Key
type Signatory [20]byte

// Equal compares two `Signatory`
func (signatory Signatory) Equal(other Signatory) bool {
	return bytes.Equal(signatory[:], other[:])
}

// Signatories is an array of Signatory
type Signatories []Signatory

// Signer signs the provided hash, returning a signature
type Signer interface {
	// Sign a `Hash` and return the resulting `Signature`.
	Sign(hash Hash) (Signature, error)

	// Signatory returns the `Signatory` of your PublicKey
	Signatory() Signatory
}

// A Verifier can return the `Signatory` that produced a `Signature`.
type Verifier interface {
	Verify(hash Hash, signature Signature) (Signatory, error)
}

// A SignerVerifier combines the `Signer` and `Verifier` interfaces.
type SignerVerifier interface {
	Signer
	Verifier
}
