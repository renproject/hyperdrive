package testutils

import (
	"crypto/rand"

	"github.com/renproject/hyperdrive/sig"
)

// RandomHash returns a random 32 byte array
func RandomHash() sig.Hash {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	hash := sig.Hash{}
	copy(hash[:], key[:])

	return hash
}

// RandomSignatory returns a random 20 byte array
func RandomSignatory() sig.Signatory {
	key := make([]byte, 20)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	signatory := sig.Signatory{}
	copy(signatory[:], key[:])

	return signatory
}

// RandomSignatories returns an array of n `sig.Signatories`
func RandomSignatories(n int) []sig.Signatory {
	signatories := []sig.Signatory{}
	for i := 0; i < n; i++ {
		signatories = append(signatories, RandomSignatory())
	}
	return signatories
}

// RandomSignature returns a random 65 byte array
func RandomSignature() sig.Signature {
	key := make([]byte, 65)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	signature := sig.Signature{}
	copy(signature[:], key[:])

	return signature
}

// RandomSignatures returns an array of n `sig.Signatures`
func RandomSignatures(n int) []sig.Signature {
	signatures := []sig.Signature{}
	for i := 0; i < n; i++ {
		signatures = append(signatures, RandomSignature())
	}
	return signatures
}
