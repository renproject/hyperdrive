package block

import (
	"fmt"

	"github.com/renproject/hyperdrive/sig"
	"golang.org/x/crypto/sha3"
)

type PreCommit struct {
	Polka Polka
}

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

type CommitBuilder map[Height]map[sig.Signatory]SignedPreCommit

func (builder CommitBuilder) Insert(preCommit SignedPreCommit) {
	if _, ok := builder[preCommit.Polka.Height]; !ok {
		builder[preCommit.Polka.Height] = map[sig.Signatory]SignedPreCommit{}
	}
	if _, ok := builder[preCommit.Polka.Height][preCommit.Signatory]; !ok {
		builder[preCommit.Polka.Height][preCommit.Signatory] = preCommit
	}
}

func (builder CommitBuilder) Commit(consensusThreshold int64) (Commit, bool) {
	highestCommitFound := false
	highestCommit := Commit{}
	for height, preCommits := range builder {
		if !highestCommitFound || height > highestCommit.Polka.Height {
			if int64(len(preCommits)) < consensusThreshold {
				continue
			}
			highestCommitFound = true

			preCommitsForNil := int64(0)
			preCommitsForBlock := map[sig.Hash]int64{}
			for _, preCommit := range preCommits {
				if preCommit.Polka.Block == nil {
					preCommitsForNil++
					continue
				}
				numPreCommits := preCommitsForBlock[preCommit.Polka.Block.Header]
				numPreCommits++
				preCommitsForBlock[preCommit.Polka.Block.Header] = numPreCommits
			}

			commitFound := false
			for blockHeader, numPreCommits := range preCommitsForBlock {
				if numPreCommits >= consensusThreshold {
					commitFound = true
					for _, preCommit := range preCommits {
						if preCommit.Polka.Block != nil && preCommit.Polka.Block.Header.Equal(blockHeader) {
							highestCommit.Polka.Block = preCommit.Polka.Block
							highestCommit.Signatories = append(highestCommit.Signatories, preCommit.Signatory)
							highestCommit.Signatures = append(highestCommit.Signatures, preCommit.Signature)
						}
					}
					break
				}
			}
			if commitFound {
				continue
			}

			for _, preCommit := range preCommits {
				if preCommit.Polka.Block == nil {
					highestCommit.Polka.Block = preCommit.Polka.Block
					highestCommit.Signatories = append(highestCommit.Signatories, preCommit.Signatory)
					highestCommit.Signatures = append(highestCommit.Signatures, preCommit.Signature)
				}
			}
		}
	}
	return highestCommit, highestCommitFound
}

func (builder CommitBuilder) Drop(fromHeight Height) {
	for height := range builder {
		if height < fromHeight {
			delete(builder, height)
		}
	}
}
