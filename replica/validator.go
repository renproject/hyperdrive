package replica

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"golang.org/x/crypto/sha3"
	"time"
)

// Validator is responsible for handling validation of blocks.
type Validator interface {
	// ValidateBlock validates a block.SignedBlock and returns true if
	// the block is valid.
	// For a SignedBlock to be valid:
	// 1. must not be nil
	// 2. have valid blockTime, Round, and Height
	// 3. have a valid signature
	// 4. signatory belongs to the same shard
	// 5. parent header is the block at head of the shard's blockchain
	ValidateBlock(signedBlock block.SignedBlock) bool

	ValidatePreVote(preVote block.SignedPreVote) bool

	ValidatePreCommit(preCommit block.SignedPreCommit) bool

	// ValidatePolka validates a polka and its signatures and returns true if
	// the polka is valid.
	// For a Polka to be valid:
	// 1. must not be nil
	// 2. have a non-negative Round and Height
	// 3. all the signatures inside the polka must be valid and
	//    belong to signatories within the same shard
	// 4. have a valid block (if block is not nil)
	//
	// validatePolka assumes that `polka.Signatures` are ordered to match
	// the order of `polka.Signatories`.
	ValidatePolka(polka block.Polka) bool
}

type validator struct {
	signer     sig.Verifier
	shard      shard.Shard
	blockchain block.Blockchain
}

// NewValidator returns a Validator
func NewValidator(signer sig.Verifier, shard shard.Shard, blockchain block.Blockchain) Validator {
	return &validator{signer, shard, blockchain}
}

func (validator *validator) ValidateBlock(signedBlock block.SignedBlock) bool {
	if signedBlock.Block.Equal(block.Block{}) {
		return false
	}
	if signedBlock.Time.After(time.Now()) {
		return false
	}
	if signedBlock.Round < 0 || signedBlock.Height < 0 {
		return false
	}

	// Verify the signatory of the signedBlock
	if !validator.verifySignature(signedBlock.Block.Header, signedBlock.Signature, signedBlock.Signatory) {
		return false
	}

	// Is block.ParentHeader (at H) == block.Header (at H-1)?
	parent, ok := validator.blockchain.Head()
	if !ok {
		return false
	}

	return parent.Header.Equal(signedBlock.ParentHeader)
}

func (validator *validator) ValidatePreVote(preVote block.SignedPreVote) bool {
	// 1. Verify the pre-vote is well-formed
	if preVote.PreVote.Block != nil {
		if !validator.ValidateBlock(*preVote.PreVote.Block) {
			return false
		}
		if preVote.PreVote.Round != preVote.PreVote.Block.Round {
			return false
		}
		if preVote.PreVote.Height != preVote.PreVote.Block.Height {
			return false
		}
	}

	// 2. Verify the signatory of the pre-vote
	hashSum256 := sha3.Sum256([]byte(preVote.PreVote.String()))
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	return validator.verifySignature(hash, preVote.Signature, preVote.Signatory)
}

func (validator *validator) ValidatePreCommit(preCommit block.SignedPreCommit) bool {
	// 1. Verify the pre-commit is well-formed
	if !validator.ValidatePolka(preCommit.PreCommit.Polka) {
		return false
	}

	// 2. Verify the signatory of the pre-commit
	hashSum256 := sha3.Sum256([]byte(preCommit.PreCommit.String()))
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	return validator.verifySignature(hash, preCommit.Signature, preCommit.Signatory)
}

func (validator *validator) ValidatePolka(polka block.Polka) bool {
	if polka.Equal(block.Polka{}) {
		return false
	}
	if polka.Round < 0 || polka.Height < 0 {
		return false
	}

	if polka.Block != nil {
		if polka.Round != polka.Block.Round {
			return false
		}
		if polka.Height != polka.Block.Height {
			return false
		}
		if !validator.ValidateBlock(*polka.Block) {
			return false
		}
	}

	preVote := block.PreVote{
		Block:  polka.Block,
		Height: polka.Height,
		Round:  polka.Round,
	}
	data := []byte(preVote.String())

	return validator.verifySignatures(data, polka.Signatures, polka.Signatories)
}

// TODO: Un-comment when needed
// func (validator *validator) ValidateCommit(commit block.Commit) bool {
// 	preCommit := block.PreCommit{
// 		Polka: commit.Polka,
// 	}
// 	data := []byte(preCommit.String())

// 	return validator.ValidatePolka(commit.Polka) && validator.verifySignatures(data, commit.Signatures, commit.Signatories)
// }

// isSignatoryInShard returns true if the given signatory belongs to the
// provided shard.
func (validator *validator) isSignatoryInShard(signatory sig.Signatory) bool {
	for _, sig := range validator.shard.Signatories {
		if signatory.Equal(sig) {
			return true
		}
	}
	return false
}

// verifySignature verifies that the signatory provided was used to generate
// the signature for the given hash. Also verifies that the signatory is a
// part of the given shard.
func (validator *validator) verifySignature(hash sig.Hash, signature sig.Signature, signatory sig.Signatory) bool {
	verifiedSig, err := validator.signer.Verify(hash, signature)
	if err != nil || !verifiedSig.Equal(signatory) {
		// TODO: log the error
		return false
	}

	return validator.isSignatoryInShard(verifiedSig)
}

func (validator *validator) verifySignatures(data []byte, signatures sig.Signatures, signatories sig.Signatories) bool {
	if len(signatories) != len(signatures) {
		return false
	}
	if len(signatories) < validator.shard.ConsensusThreshold() {
		return false
	}
	hashSum256 := sha3.Sum256(data)
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	for i, signature := range signatures {
		if !validator.verifySignature(hash, signature, signatories[i]) {
			return false
		}
	}
	return true
}
