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

func (preVote PreVote) Sign(signer sig.Signer) (SignedPreVote, error) {
	data := []byte(preVote.String())

	hashSum256 := sha3.Sum256(data)
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	signature, err := signer.Sign(hash)
	if err != nil {
		return SignedPreVote{}, err
	}

	return SignedPreVote{
		PreVote:   preVote,
		Signature: signature,
		Signatory: signer.Signatory(),
	}, nil
}

func (preVote PreVote) String() string {
	block := "Nil"
	if preVote.Block != nil {
		block = preVote.Block.String()
	}
	return fmt.Sprintf("PreVote(%s,Round=%d,Height=%d)", block, preVote.Round, preVote.Height)
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

// Polka finds the `Polka` with the highest height that has enough
// `PreVotes` to be greater than or equal to the `consensusThreshold`
func (builder PolkaBuilder) Polka(consensusThreshold int64) (Polka, bool) {
	highestPolkaFound := false
	highestPolka := Polka{}
	for height, preVotes := range builder {
		if !highestPolkaFound || height > highestPolka.Block.Height {
			if int64(len(preVotes)) < consensusThreshold {
				continue
			}
			highestPolkaFound = true

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

			polkaFound := false
			for blockHeader, numPreVotes := range preVotesForBlock {
				if numPreVotes >= consensusThreshold {
					polkaFound = true
					for _, preVote := range preVotes {
						if preVote.Block != nil &&
							preVote.Block.Header.Equal(blockHeader) {

							highestPolka.Block = preVote.Block
							highestPolka.Round = preVote.Round
							highestPolka.Height = preVote.Height
							highestPolka.Signatories =
								append(highestPolka.Signatories,
									preVote.Signatory)
							highestPolka.Signatures =
								append(highestPolka.Signatures,
									preVote.Signature)
						}
					}
					break
				}
			}
			if polkaFound {
				continue
			}

			for _, preVote := range preVotes {
				if preVote.Block == nil {
					highestPolka.Block = preVote.Block
					highestPolka.Round = preVote.Round
					highestPolka.Height = preVote.Height
					highestPolka.Signatories =
						append(highestPolka.Signatories, preVote.Signatory)
					highestPolka.Signatures =
						append(highestPolka.Signatures, preVote.Signature)
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
