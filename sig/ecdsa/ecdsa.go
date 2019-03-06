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

func (signerVerifier signerVerifier) Sign(hash sig.Hash) (sig.Signature, error) {
	signed, err := crypto.Sign(hash[:], signerVerifier.privKey)
	sig := sig.Signature{}
	copy(sig[:], signed)
	return sig, err
}

func (signerVerifier signerVerifier) Signatory() sig.Signatory {
	return PubKeyToSignatory(&signerVerifier.privKey.PublicKey)
}

func (signerVerifier signerVerifier) Verify(hash sig.Hash,
	signature sig.Signature) (sig.Signatory, error) {
	pubKey, err := crypto.SigToPub(hash[:], signature[:])
	return PubKeyToSignatory(pubKey), err
}
