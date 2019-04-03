// Package ecdsa provides Sign and Verify capabilities using ECDSA
//
// I picked ethereum-go's implementation of ECDSA instead of the built in
// go libraries since it provides both build in marshaling/unmarshaling
// for the signature and a way to recover the Public Key from the
// signature. The reality is, we don't want to check if a given signature
// matches a given Public Key. Instead we want to know who signed this
// message, which is the Public Key. Then its easy to write logic to
// ensure nobody votes twice given a list of Public Keys or a list of
// hashed Public Keys, etc...
package ecdsa

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/renproject/hyperdrive/sig"
)

// PubKeyToSignatory returns a `Signatory` using the same conversion
// that Ethereum uses to extract addresses from public keys.
func PubKeyToSignatory(pubKey *ecdsa.PublicKey) sig.Signatory {
	bytes := elliptic.Marshal(secp256k1.S256(), pubKey.X, pubKey.Y)
	hash := crypto.Keccak256(bytes[1:]) // Keccak256 hash
	hash = hash[(len(hash) - 20):]      // Take the last 20 bytes
	data := sig.Signatory{}
	copy(data[:], hash)
	return data
}

// Hash runs Keccak256 and returns a Hash
func Hash(data []byte) sig.Hash {
	hash := crypto.Keccak256(data) // Keccak256 hash
	hashed := sig.Hash{}
	copy(hashed[:], hash)
	return hashed
}

// signerVerifier provides Sign and Verify capabilities using ECDSA
//
// Ethereum-go's ECDSA implementation was chosen instead of the built-in go
// library since it provides both built-in marshaling/unmarshaling for the
// signature and a way to recover the Public Key from the signature, which
// can be used to identify the party that signed a message.
type signerVerifier struct {
	privKey *ecdsa.PrivateKey
}

// NewFromPrivKey takes a provided Private Key
func NewFromPrivKey(privKey *ecdsa.PrivateKey) sig.SignerVerifier {
	return signerVerifier{
		privKey: privKey,
	}
}

// NewFromRandom generates a new random Private Key
func NewFromRandom() (sig.SignerVerifier, error) {
	privKey, err := crypto.GenerateKey()
	return signerVerifier{
		privKey: privKey,
	}, err
}

// Sign implements the `sig.SignerVerifier` interface
func (signerVerifier signerVerifier) Sign(hash sig.Hash) (sig.Signature, error) {
	signed, err := crypto.Sign(hash[:], signerVerifier.privKey)
	sig := sig.Signature{}
	copy(sig[:], signed)
	return sig, err
}

// Signatory implements the `sig.SignerVerifier` interface
func (signerVerifier signerVerifier) Signatory() sig.Signatory {
	return PubKeyToSignatory(&signerVerifier.privKey.PublicKey)
}

// Verify implements the `sig.SignerVerifier` interface
func (signerVerifier signerVerifier) Verify(hash sig.Hash,
	signature sig.Signature) (sig.Signatory, error) {
	pubKey, err := crypto.SigToPub(hash[:], signature[:])
	return PubKeyToSignatory(pubKey), err
}
