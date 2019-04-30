package testutils

import (
	"fmt"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
)

func SignBlock(blk block.Block, signer sig.SignerVerifier) *block.SignedBlock {
	signedBlock, err := blk.Sign(signer)
	if err != nil {
		panic(fmt.Sprintf("error signing block: %v", err))
	}
	return &signedBlock
}

func GenerateSignedPreVote(signedBlock block.SignedBlock, signer sig.SignerVerifier) block.SignedPreVote {
	preVote := block.PreVote{
		Block:  &signedBlock,
		Height: signedBlock.Height,
	}
	signedPreVote, err := preVote.Sign(signer)
	if err != nil {
		panic(fmt.Sprintf("error signing preVote: %v", err))
	}
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

	signedPreCommit, err := preCommit.Sign(signer)
	if err != nil {
		panic(fmt.Sprintf("error signing preCommit: %v", err))
	}
	return signedPreCommit
}
