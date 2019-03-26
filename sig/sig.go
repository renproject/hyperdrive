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

// Equal compares two `Signatory`
func (signature Signature) Equal(other Signature) bool {
	return bytes.Equal(signature[:], other[:])
}

// Signatures is an array of Signature
type Signatures []Signature

// Equal checks for set equality of Signatures, order does not matter
func (sig Signatures) Equal(other Signatures) bool {
	if len(sig) != len(other) {
		return false
	}
	// create a map of string -> int
	diff := make(map[Signature]int, len(sig))
	for _, _x := range sig {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range other {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	if len(diff) == 0 {
		return true
	}
	return false
}

// Signatory is the last 20 bytes of a Public Key
type Signatory [20]byte

// Equal compares two `Signatory`
func (signatory Signatory) Equal(other Signatory) bool {
	return bytes.Equal(signatory[:], other[:])
}

// Signatories is an array of Signatory
type Signatories []Signatory

// Equal checks for set equality of Signatories, order does not matter
func (sig Signatories) Equal(other Signatories) bool {
	if len(sig) != len(other) {
		return false
	}
	// create a map of string -> int
	diff := make(map[Signatory]int, len(sig))
	for _, _x := range sig {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range other {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	if len(diff) == 0 {
		return true
	}
	return false
}

// Signer signs the provided hash, returning a signature
type Signer interface {
	// Sign a `Hash` and return the resulting `Signature`.
	Sign(hash Hash) (Signature, error)

	// Signatory returns the `Signatory` of your PublicKey
	Signatory() Signatory
}

// A Verifier can return the `Signatory` that produced a `Signature`.
type Verifier interface {
	// Note: Verify will not return an error if the hash and signature
	// do not match, but the returned public key is effectively random.
	Verify(hash Hash, signature Signature) (Signatory, error)
}

// A SignerVerifier combines the `Signer` and `Verifier` interfaces.
type SignerVerifier interface {
	Signer
	Verifier
}
