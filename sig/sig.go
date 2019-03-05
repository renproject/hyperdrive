package sig

import (
	"bytes"
)

type Hash [32]byte

func (hash Hash) Equal(other Hash) bool {
	return bytes.Equal(hash[:], other[:])
}

type Signature []byte

type Signatures []Signature

type Signatory [20]byte

type Signatories []Signatory

// Signer signs the provided hash, returning a signature
type Signer interface {
	// Sign a `Hash` and return the resulting `Signature`.
	Sign(hash Hash) (Signature, error)

	// Signatory returns the `Signatory` that is returned when verifying a `Signature`.
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
