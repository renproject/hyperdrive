package ecdsa

import (
	"crypto/ecdsa"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/sig"
)

// New needs who you are and your private ECDSA key
func New(whoami sig.Signatory,
	privKey *ecdsa.PrivateKey) sig.Authenticator {
	return signer{
		whoami:  whoami,
		privKey: privKey,
	}
}

// NewRandKey Generates a private key for you
func NewRandKey(whoami sig.Signatory) (sig.Authenticator, error) {
	privKey, err := ethCrypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return signer{
		whoami:  whoami,
		privKey: privKey,
	}, nil
}

type signer struct {
	whoami  sig.Signatory
	privKey *ecdsa.PrivateKey
}

func (signer signer) Sign(hash sig.Hash) (sig.Signature, sig.Signatory, error) {
	sig, err := ethCrypto.Sign(hash[:], signer.privKey)

	if err != nil {
		return nil, signer.whoami, err
	}

	return sig, signer.whoami, err
}

func (signer signer) PubKey() *ecdsa.PublicKey {
	return &signer.privKey.PublicKey
}

func (signer signer) Verify(hash sig.Hash, sig sig.Signature) (*ecdsa.PublicKey, error) {
	return ethCrypto.SigToPub(hash[:], sig)
}
