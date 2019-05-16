package block

import (
	"fmt"

	"github.com/renproject/hyperdrive/sig"
	"golang.org/x/crypto/sha3"
)

type PreCommit struct {
	Polka Polka
}

// Sign a PreCommit with your private key
func (preCommit PreCommit) Sign(signer sig.Signer) (SignedPreCommit, error) {
	data := []byte(preCommit.String())

	hashSum256 := sha3.Sum256(data)
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	signature, err := signer.Sign(hash)
	if err != nil {
		return SignedPreCommit{}, err
	}

	return SignedPreCommit{
		PreCommit: preCommit,
		Signature: signature,
		Signatory: signer.Signatory(),
	}, nil
}

func (preCommit PreCommit) String() string {
	return fmt.Sprintf("PreCommit(%s)", preCommit.Polka.String())
}

type SignedPreCommit struct {
	PreCommit
	Signature sig.Signature
	Signatory sig.Signatory
}

type Commit struct {
	Polka       Polka
	Signatures  sig.Signatures
	Signatories sig.Signatories
}

func (commit Commit) String() string {
	return fmt.Sprintf("Commit(%s)", commit.Polka.String())
}

// CommitBuilder is used to build up collections of SignedPreCommits at different Heights and Rounds and then build
// Commits wherever there are enough SignedPreCommits to do so.
type CommitBuilder map[Height]map[Round]map[sig.Signatory]SignedPreCommit

func NewCommitBuilder() CommitBuilder {
	return CommitBuilder{}
}

// Insert a SignedPreCommit into the CommitBuilder. This will include the SignedPreCommit in all attempts to build a
// Commit for the respective Height.
func (builder CommitBuilder) Insert(preCommit SignedPreCommit) bool {
	// Pre-condition check
	if preCommit.Polka.Block != nil {
		if preCommit.Polka.Block.Height != preCommit.Polka.Height {
			panic(fmt.Errorf("expected pre-commit height (%v) to equal pre-commit block height (%v)", preCommit.Polka.Height, preCommit.Polka.Block.Height))
		}
		if preCommit.Polka.Block.Round != preCommit.Polka.Round {
			panic(fmt.Errorf("expected pre-commit round (%v) to equal pre-commit block round (%v)", preCommit.Polka.Round, preCommit.Polka.Block.Round))
		}
	}

	if _, ok := builder[preCommit.Polka.Height]; !ok {
		builder[preCommit.Polka.Height] = map[Round]map[sig.Signatory]SignedPreCommit{}
	}
	if _, ok := builder[preCommit.Polka.Height][preCommit.Polka.Round]; !ok {
		builder[preCommit.Polka.Height][preCommit.Polka.Round] = map[sig.Signatory]SignedPreCommit{}
	}
	if _, ok := builder[preCommit.Polka.Height][preCommit.Polka.Round][preCommit.Signatory]; !ok {
		builder[preCommit.Polka.Height][preCommit.Polka.Round][preCommit.Signatory] = preCommit
		return true
	}
	return false
}

// Commit returns a Commit and the latest Round for which there are 2/3rd+
// pre-commits. The Commit will be nil unless there is 2/3rds+ pre-commits for
// nil or a specific SignedBlock. The Round will be nil unless there are 2/3rds+
// pre-commits for a specific Round. If a Commit is returned, the Round will
// match the Commit.
func (builder CommitBuilder) Commit(height Height, consensusThreshold int) (*Commit, *Round) {
	// Pre-condition check
	if consensusThreshold < 1 {
		panic(fmt.Errorf("expected consensus threshold (%v) to be greater than 0", consensusThreshold))
	}

	// Short-circuit when too few pre-commits have been received
	preCommitsByRound, ok := builder[height]
	if !ok {
		return nil, nil
	}

	var commit *Commit
	var preCommitsRound *Round

	for round, preCommits := range preCommitsByRound {

		numNilPreCommits := 0
		nilPreCommitSignatures := []sig.Signature{}
		nilPreCommitSignatories := []sig.Signatory{}

		if commit != nil && round <= commit.Polka.Round {
			continue
		}
		if len(preCommits) < consensusThreshold {
			continue
		}
		preCommitsRound = &round

		// Build a mapping of the pre-commits for each block
		preCommitsForBlock := map[sig.Hash]int{}
		for _, preCommit := range preCommits {
			// Invariant check
			if preCommit.Polka.Height != height {
				panic(fmt.Errorf("expected pre-commit height (%v) to equal %v", preCommit.Polka.Height, height))
			}
			if preCommit.Polka.Round != round {
				panic(fmt.Errorf("expected pre-commit round (%v) to equal %v", preCommit.Polka.Round, round))
			}
			if preCommit.Polka.Block == nil {
				numNilPreCommits++
				nilPreCommitSignatures = append(nilPreCommitSignatures, preCommit.Signature)
				nilPreCommitSignatories = append(nilPreCommitSignatories, preCommit.Signatory)
				continue
			}

			// Invariant check
			if preCommit.Polka.Block.Height != height {
				panic(fmt.Errorf("expected pre-commit block height (%v) to equal %v", preCommit.Polka.Block.Height, height))
			}
			if preCommit.Polka.Block.Round != round {
				panic(fmt.Errorf("expected pre-commit block round (%v) to equal %v", preCommit.Polka.Block.Round, round))
			}
			preCommitsForBlock[preCommit.Polka.Block.Header]++
		}

		// Search for a commit of pre-commits for non-nil block
		for blockHeader, numPreVotes := range preCommitsForBlock {
			if numPreVotes >= consensusThreshold {
				commit = &Commit{
					Signatures:  make(sig.Signatures, 0, consensusThreshold),
					Signatories: make(sig.Signatories, 0, consensusThreshold),
				}
				for _, preCommit := range preCommits {
					if preCommit.Polka.Block != nil && preCommit.Polka.Block.Header.Equal(blockHeader) {
						if commit.Polka.Block != nil {
							// Invariant check
							if commit.Polka.Round != preCommit.Polka.Round {
								panic(fmt.Errorf("expected commit round (%v) to equal pre-commit round (%v)", commit.Polka.Round, preCommit.Polka.Round))
							}
							if commit.Polka.Height != preCommit.Polka.Height {
								panic(fmt.Errorf("expected commit height (%v) to equal pre-commit height (%v)", commit.Polka.Height, preCommit.Polka.Height))
							}
						} else {
							// Invariant check
							if preCommit.Polka.Height != height {
								panic(fmt.Errorf("expected pre-commit height (%v) to equal %v", preCommit.Polka.Height, height))
							}
							if preCommit.Polka.Round != round {
								panic(fmt.Errorf("expected pre-commit round (%v) to equal %v", preCommit.Polka.Round, round))
							}
							commit.Polka = preCommit.Polka
						}
						commit.Signatures = append(commit.Signatures, preCommit.Signature)
						commit.Signatories = append(commit.Signatories, preCommit.Signatory)
					}
				}
				break
			}
		}

		if numNilPreCommits >= consensusThreshold {
			// Return a nil-Commit
			commit = &Commit{
				Polka: Polka{
					Block:  nil,
					Height: height,
					Round:  round,
				},
				Signatures:  nilPreCommitSignatures,
				Signatories: nilPreCommitSignatories,
			}
		}
	}

	if commit != nil {
		// Post-condition check
		if commit.Polka.Block != nil {
			if len(commit.Signatures) != len(commit.Signatories) {
				panic(fmt.Errorf("expected the number of signatures (%v) to be equal to the number of signatories (%v)", len(commit.Signatures), len(commit.Signatories)))
			}
			if len(commit.Signatures) < consensusThreshold {
				panic(fmt.Errorf("expected the number of signatures (%v) to be greater than or equal to the consensus threshold (%v)", len(commit.Signatures), consensusThreshold))
			}
		}
		if commit.Polka.Height != height {
			panic(fmt.Errorf("expected the commit height (%v) to equal %v", commit.Polka.Height, height))
		}
		return commit, &commit.Polka.Round
	}
	return nil, preCommitsRound
}

func (builder CommitBuilder) Drop(fromHeight Height) {
	for height := range builder {
		if height < fromHeight {
			delete(builder, height)
		}
	}
}
