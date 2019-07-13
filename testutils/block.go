package testutils

import (
	"fmt"
	"math/rand"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/tx"
)

func GenerateSignedBlock() (block.SignedBlock, sig.SignerVerifier, error) {
	signer, err := ecdsa.NewFromRandom()
	if err != nil {
		return block.SignedBlock{}, nil, err
	}

	block := block.New(block.Height(rand.Int()), RandomHash(), []tx.Transaction{RandomTransaction()})
	return *SignBlock(block, signer), signer, nil
}

func SignBlock(blk block.Block, signer sig.SignerVerifier) *block.SignedBlock {
	signedBlock, err := blk.Sign(signer)
	if err != nil {
		panic(fmt.Sprintf("error signing block: %v", err))
	}
	return &signedBlock
}

func GenerateSignedPropose(signedBlock block.SignedBlock, round block.Round, signer sig.SignerVerifier) block.SignedPropose {
	propose := block.Propose{
		Block: signedBlock,
		Round: round,
	}
	signedPropose, err := propose.Sign(signer)
	if err != nil {
		panic(fmt.Sprintf("error signing preVote: %v", err))
	}
	return signedPropose
}

func GenerateSignedPreVote(signedBlock *block.SignedBlock, signer sig.SignerVerifier) block.SignedPreVote {
	height := block.Height(0)
	if signedBlock != nil {
		height = signedBlock.Height
	}
	preVote := block.PreVote{
		Block:  signedBlock,
		Height: height,
	}
	signedPreVote, err := preVote.Sign(signer)
	if err != nil {
		panic(fmt.Sprintf("error signing preVote: %v", err))
	}
	return signedPreVote
}

func GeneratePolkaWithSignatures(signedBlock *block.SignedBlock, participants []sig.SignerVerifier) block.Polka {
	signatures := sig.Signatures{}
	signatories := sig.Signatories{}
	for _, participant := range participants {
		signedPreVote := GenerateSignedPreVote(signedBlock, participant)
		signatures = append(signatures, signedPreVote.Signature)
		signatories = append(signatories, signedPreVote.Signatory)
	}
	height := block.Height(0)
	if signedBlock != nil {
		height = signedBlock.Height
	}
	return block.Polka{
		Block:       signedBlock,
		Height:      height,
		Signatures:  signatures,
		Signatories: signatories,
	}
}

func GenerateSignedPreCommit(signedBlock block.SignedBlock, signer sig.SignerVerifier, participants []sig.SignerVerifier) block.SignedPreCommit {
	preCommit := block.PreCommit{
		Polka: GeneratePolkaWithSignatures(&signedBlock, participants),
	}

	signedPreCommit, err := preCommit.Sign(signer)
	if err != nil {
		panic(fmt.Sprintf("error signing preCommit: %v", err))
	}
	return signedPreCommit
}
