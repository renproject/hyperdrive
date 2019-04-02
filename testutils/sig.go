package testutils

import (
	"crypto/rand"

	"github.com/renproject/hyperdrive/block"
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

func SignBlock(blk block.Block, signer sig.SignerVerifier) block.SignedBlock {
	signedBlock, _ := blk.Sign(signer)
	return signedBlock
}

func GenerateSignedPreVote(signedBlock block.SignedBlock, signer sig.SignerVerifier) block.SignedPreVote {
	preVote := block.PreVote{
		Block:  &signedBlock,
		Height: signedBlock.Height,
	}
	signedPreVote, _ := preVote.Sign(signer)
	return signedPreVote
}

func GeneratePolkaWithSignatures(signedBlock block.SignedBlock, participants []sig.SignerVerifier) block.Polka {
	signatures := sig.Signatures{}
	signatories := sig.Signatories{}
	for _, participant := range participants {
		signedPreVote := GenerateSignedPreVote(signedBlock, participant)
		signatures = append(signatures, signedPreVote.Signature)
		signatories = append(signatories, signedPreVote.Signatory)
	}

	return block.Polka{
		Block:       &signedBlock,
		Height:      signedBlock.Height,
		Signatures:  signatures,
		Signatories: signatories,
	}
}

func GenerateSignedPreCommit(signedBlock block.SignedBlock, signer sig.SignerVerifier, participants []sig.SignerVerifier) block.SignedPreCommit {
	preCommit := block.PreCommit{
		Polka: GeneratePolkaWithSignatures(signedBlock, participants),
	}

	signedPreCommit, _ := preCommit.Sign(signer)
	return signedPreCommit
}
