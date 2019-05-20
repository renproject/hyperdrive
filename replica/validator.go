package replica

import (
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"golang.org/x/crypto/sha3"
)

// Validator is responsible for handling validation of blocks.
type Validator interface {
	// ValidatePropose validates a state.Propose and returns true if
	// the block is valid.
	// For a SignedBlock to be valid:
	// 1. must not be nil
	// 2. have valid blockTime, Round, and Height
	// 3. have a valid signature
	// 4. signatory belongs to the same shard
	// 5. parent header is the block at head of the shard's blockchain
	ValidatePropose(propose block.SignedPropose, lastSignedBlock *block.SignedBlock) bool

	ValidatePreVote(preVote block.SignedPreVote, lastSignedBlock *block.SignedBlock) bool

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
	ValidatePolka(polka block.Polka, lastSignedBlock *block.SignedBlock) bool

	ValidatePreCommit(preCommit block.SignedPreCommit, lastSignedBlock *block.SignedBlock) bool

	ValidateCommit(commit block.Commit) bool
}

type validator struct {
	signer sig.Verifier
	shard  shard.Shard

	verifiedSignatureCache map[sig.Hash]map[sig.Signature]sig.Signatory
}

// NewValidator returns a Validator
func NewValidator(signer sig.Verifier, shard shard.Shard) Validator {
	return &validator{
		signer: signer,
		shard:  shard,

		verifiedSignatureCache: map[sig.Hash]map[sig.Signature]sig.Signatory{},
	}
}

func (validator *validator) ValidatePropose(propose block.SignedPropose, lastSignedBlock *block.SignedBlock) bool {
	if propose.Round < 0 {
		return false
	}

	// Compute the PreVote hash
	hashSum256 := sha3.Sum256([]byte(propose.String()))
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	// TODO: Check cache

	// Verify the signature
	if !validator.verifySignature(hash, propose.Signature, propose.Signatory) {
		return false
	}

	return validator.ValidateBlock(propose.Block, lastSignedBlock)
}

func (validator *validator) ValidateBlock(signedBlock block.SignedBlock, lastSignedBlock *block.SignedBlock) bool {
	if signedBlock.Block.Equal(block.Block{}) {
		return false
	}
	if signedBlock.Time.After(time.Now()) {
		return false
	}
	if signedBlock.Height < 0 {
		return false
	}

	// TODO: Verify the Block header equals the expected header.

	// Verify the parent block
	if lastSignedBlock != nil {
		if !lastSignedBlock.Header.Equal(signedBlock.ParentHeader) {
			return false
		}
	}

	// TODO: Check cache

	// Verify the signature
	if !validator.verifySignature(signedBlock.Block.Header, signedBlock.Signature, signedBlock.Signatory) {
		return false
	}

	// TODO: Fill cache
	return true
}

func (validator *validator) ValidatePreVote(preVote block.SignedPreVote, lastSignedBlock *block.SignedBlock) bool {
	// Verify the pre-vote is well-formed
	if preVote.PreVote.Block != nil {
		if !validator.ValidateBlock(*preVote.PreVote.Block, lastSignedBlock) {
			return false
		}
		if preVote.PreVote.Height != preVote.PreVote.Block.Height {
			return false
		}
	}
	if preVote.PreVote.Round < 0 || preVote.PreVote.Height < 0 {
		return false
	}

	// Compute the PreVote hash
	hashSum256 := sha3.Sum256([]byte(preVote.PreVote.String()))
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	// TODO: Check cache

	// Verify the signature
	if !validator.verifySignature(hash, preVote.Signature, preVote.Signatory) {
		return false
	}

	// TODO: Fill cache
	return true
}

func (validator *validator) ValidatePolka(polka block.Polka, lastSignedBlock *block.SignedBlock) bool {
	if polka.Equal(&block.Polka{}) {
		return false
	}
	if polka.Round < 0 || polka.Height < 0 {
		return false
	}

	if polka.Block != nil {
		if polka.Height != polka.Block.Height {
			return false
		}
		if !validator.ValidateBlock(*polka.Block, lastSignedBlock) {
			return false
		}
	}

	preVote := block.PreVote{
		Block:  polka.Block,
		Height: polka.Height,
		Round:  polka.Round,
	}
	data := []byte(preVote.String())

	// TODO: Check cache

	// Verify the signature
	if !validator.verifySignatures(data, polka.Signatures, polka.Signatories) {
		return false
	}

	// TODO: Fill cache
	return true
}

func (validator *validator) ValidatePreCommit(preCommit block.SignedPreCommit, lastSignedBlock *block.SignedBlock) bool {
	// Verify the underlying Polka is well-formed
	if !validator.ValidatePolka(preCommit.PreCommit.Polka, lastSignedBlock) {
		return false
	}

	// Compute the PreCommit hash
	hashSum256 := sha3.Sum256([]byte(preCommit.PreCommit.String()))
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	// TODO: Check cache

	// Verify the signature
	if !validator.verifySignature(hash, preCommit.Signature, preCommit.Signatory) {
		return false
	}

	// TODO: Fill cache
	return true
}

func (validator *validator) ValidateCommit(commit block.Commit) bool {
	preCommit := block.PreCommit{
		Polka: commit.Polka,
	}
	data := []byte(preCommit.String())

	return validator.ValidatePolka(commit.Polka, nil) && validator.verifySignatures(data, commit.Signatures, commit.Signatories)
}

// verifySignature verifies that the signatory provided was used to generate
// the signature for the given hash. Also verifies that the signatory is a
// part of the given shard.
func (validator *validator) verifySignature(hash sig.Hash, signature sig.Signature, signatory sig.Signatory) bool {
	// Short-circuit using the cache
	if cachedSignatures, ok := validator.verifiedSignatureCache[hash]; ok {
		if cachedSignatory, ok := cachedSignatures[signature]; ok {
			if cachedSignatory.Equal(signatory) {
				return true
			}
		}
	}

	verifiedSignatory, err := validator.signer.Verify(hash, signature)
	if err != nil || !verifiedSignatory.Equal(signatory) {
		// FIXME: Do we need to log the error?
		return false
	}

	if !validator.isSignatoryInShard(verifiedSignatory) {
		return false
	}

	// Fill the cache
	if cachedSignatures, ok := validator.verifiedSignatureCache[hash]; ok {
		cachedSignatures[signature] = verifiedSignatory
	} else {
		validator.verifiedSignatureCache[hash] = map[sig.Signature]sig.Signatory{
			signature: verifiedSignatory,
		}
	}
	return true
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
