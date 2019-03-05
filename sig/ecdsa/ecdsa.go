package ecdsa

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/renproject/hyperdrive/sig"
)

// PubKeyToSignatory returns a `Signatory` using the same conversion that Ethereum uses to extract addresses from public
// keys.
func PubKeyToSignatory(pubKey *ecdsa.PublicKey) sig.Signatory {
	bytes := elliptic.Marshal(secp256k1.S256(), pubKey.X, pubKey.Y)
	hash := crypto.Keccak256(bytes[1:]) // Keccak256 hash
	hash = hash[(len(hash) - 20):]      // Take the last 20 bytes
	data := sig.Signatory{}
	copy(data[:], hash)
	return data
}

type signerVerifier struct {
	privKey *ecdsa.PrivateKey
}

func NewFromPrivKey(privKey *ecdsa.PrivateKey) sig.SignerVerifier {
	return signerVerifier{
		privKey: privKey,
	}
}

func NewFromRandom() (sig.SignerVerifier, error) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return signerVerifier{
		privKey: privKey,
	}, nil
}

func (signerVerifier signerVerifier) Sign(hash sig.Hash) (sig.Signature, error) {
	sig, err := crypto.Sign(hash[:], signerVerifier.privKey)
	return sig, err
}

func (signerVerifier signerVerifier) Signatory() sig.Signatory {
	return PubKeyToSignatory(&signerVerifier.privKey.PublicKey)
}

func (signerVerifier signerVerifier) Verify(hash sig.Hash, signature sig.Signature) (sig.Signatory, error) {
	pubKey, err := crypto.SigToPub(hash[:], signature)
	if err != nil {
		return sig.Signatory{}, err
	}
	return PubKeyToSignatory(pubKey), nil
}
