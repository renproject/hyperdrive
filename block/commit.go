package block

import (
	"fmt"

	"github.com/renproject/hyperdrive/sig"
	"golang.org/x/crypto/sha3"
)

const PrefixPreCommit = "PreCommit: "

type PreCommit struct {
	Polka Polka
}

func (preCommit PreCommit) Sign(signer sig.Signer) SignedPreCommit {
	data := []byte(PrefixPreCommit)
	if preCommit.Polka.Block != nil {
		data = append(data, preCommit.Polka.Block.Header[:]...)
	} else {
		data = append(data, []byte("Nil")...)
	}
	data = append(data, []byte(fmt.Sprintf(" Round: %d", preCommit.Polka.Round))...)
	data = append(data, []byte(fmt.Sprintf(" Height: %d", preCommit.Polka.Height))...)

	hashSum256 := sha3.Sum256(data)
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])
	signature, signatory := signer.Sign(hash)

	return SignedPreCommit{
		PreCommit: preCommit,
		Signature: signature,
		Signatory: signatory,
	}
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
	if _, ok := builder[preCommit.Polka.Block.Height]; !ok {
		builder[preCommit.Polka.Block.Height] = map[sig.Signatory]SignedPreCommit{}
	}
	if _, ok := builder[preCommit.Polka.Block.Height][preCommit.Signatory]; !ok {
		builder[preCommit.Polka.Block.Height][preCommit.Signatory] = preCommit
	}
}

func (builder CommitBuilder) Commit(consensusThreshold int64) (Commit, bool) {
	highestCommitFound := false
	highestCommit := Commit{}
	for height, preCommits := range builder {
		if !highestCommitFound || height > highestCommit.Polka.Block.Height {

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

			if preCommitsForNil >= consensusThreshold {
				highestCommitFound = true
				for _, preCommit := range preCommits {
					if preCommit.Polka.Block == nil {
						highestCommit.Polka.Block = preCommit.Polka.Block
						highestCommit.Signatories = append(highestCommit.Signatories, preCommit.Signatory)
						highestCommit.Signatures = append(highestCommit.Signatures, preCommit.Signature)
					}
				}
			}

			for blockHeader, numPreCommits := range preCommitsForBlock {
				if numPreCommits >= consensusThreshold {
					highestCommitFound = true
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
