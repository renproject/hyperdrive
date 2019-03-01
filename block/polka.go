package block

import (
	"encoding/base64"
	"fmt"

	"github.com/renproject/hyperdrive/sig"
	"golang.org/x/crypto/sha3"
)

type PreVote struct {
	Block  *Block
	Round  Round
	Height Height
}

func NewPreVote(block *Block, round Round, height Height) PreVote {
	return PreVote{
		Block:  block,
		Round:  round,
		Height: height,
	}
}

func (preVote PreVote) Sign(signer sig.Signer) SignedPreVote {
	data := []byte(preVote.String())

	hashSum256 := sha3.Sum256(data)
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])
	signature, signatory := signer.Sign(hash)

	return SignedPreVote{
		PreVote:   preVote,
		Signature: signature,
		Signatory: signatory,
	}
}

func (preVote PreVote) String() string {
	blockHeader := "Nil"
	if preVote.Block != nil {
		blockHeader = base64.StdEncoding.EncodeToString(preVote.Block.Header[:])
	}
	return fmt.Sprintf("PreVote(Block=%s,Round=%d,Height=%d)", blockHeader, preVote.Round, preVote.Height)
}

type SignedPreVote struct {
	PreVote
	Signature sig.Signature
	Signatory sig.Signatory
}

type Polka struct {
	Block       *Block
	Round       Round
	Height      Height
	Signatures  sig.Signatures
	Signatories sig.Signatories
}

func (polka Polka) String() string {
	blockHeader := "Nil"
	if polka.Block != nil {
		blockHeader = base64.StdEncoding.EncodeToString(polka.Block.Header[:])
	}
	return fmt.Sprintf("Polka(Block=%s,Round=%d,Height=%d)", blockHeader, polka.Round, polka.Height)
}

type PolkaBuilder map[Height]map[sig.Signatory]SignedPreVote

func (builder PolkaBuilder) Insert(preVote SignedPreVote) {
	if _, ok := builder[preVote.Block.Height]; !ok {
		builder[preVote.Block.Height] = map[sig.Signatory]SignedPreVote{}
	}
	if _, ok := builder[preVote.Block.Height][preVote.Signatory]; !ok {
		builder[preVote.Block.Height][preVote.Signatory] = preVote
	}
}

func (builder PolkaBuilder) Polka(consensusThreshold int64) (Polka, bool) {
	highestPolkaFound := false
	highestPolka := Polka{}
	for height, preVotes := range builder {
		if !highestPolkaFound || height > highestPolka.Block.Height {

			preVotesForNil := int64(0)
			preVotesForBlock := map[sig.Hash]int64{}

			for _, preVote := range preVotes {
				if preVote.Block == nil {
					preVotesForNil++
					continue
				}
				numPreVotes := preVotesForBlock[preVote.Block.Header]
				numPreVotes++
				preVotesForBlock[preVote.Block.Header] = numPreVotes
			}

			if preVotesForNil >= consensusThreshold {
				highestPolkaFound = true
				for _, preVote := range preVotes {
					if preVote.Block == nil {
						highestPolka.Block = preVote.Block
						highestPolka.Round = preVote.Round
						highestPolka.Height = preVote.Height
						highestPolka.Signatories = append(highestPolka.Signatories, preVote.Signatory)
						highestPolka.Signatures = append(highestPolka.Signatures, preVote.Signature)
					}
				}
			}

			for blockHeader, numPreVotes := range preVotesForBlock {
				if numPreVotes >= consensusThreshold {
					highestPolkaFound = true
					for _, preVote := range preVotes {
						if preVote.Block != nil && preVote.Block.Header.Equal(blockHeader) {
							highestPolka.Block = preVote.Block
							highestPolka.Round = preVote.Round
							highestPolka.Height = preVote.Height
							highestPolka.Signatories = append(highestPolka.Signatories, preVote.Signatory)
							highestPolka.Signatures = append(highestPolka.Signatures, preVote.Signature)
						}
					}
					break
				}
			}
		}
	}
	return highestPolka, highestPolkaFound
}

func (builder PolkaBuilder) Drop(dropHeight Height) {
	for height := range builder {
		if height < dropHeight {
			delete(builder, height)
		}
	}
}
