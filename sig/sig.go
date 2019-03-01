package sig

import "bytes"

type Hash [32]byte

func (hash Hash) Equal(other Hash) bool {
	return bytes.Equal(hash[:], other[:])
}

type Signature [65]byte

type Signatures []Signature

type Signatory [40]byte

type Signatories []Signatory

type Signer interface {
	Sign(hash Hash) (Signature, Signatory)
}
