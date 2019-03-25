package block

import (
	"encoding/base64"
	"fmt"

	"github.com/renproject/hyperdrive/sig"
	"golang.org/x/crypto/sha3"
)

// PreVote is you voting for the encapsulated block to be added
// during this round at this height
type PreVote struct {
	Block  *Block
	Round  Round
	Height Height
}

// NewPreVote creates a PreVote
func NewPreVote(block *Block, round Round, height Height) PreVote {
	return PreVote{
		Block:  block,
		Round:  round,
		Height: height,
	}
}

// Sign is indented to be the way you sign your PreVote with your
// private key
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

// SignedPreVote is the signed version of a PreVote
type SignedPreVote struct {
	PreVote
	Signature sig.Signature
	Signatory sig.Signatory
}

// Polka is created when 2/3 of the nodes in the network have all
// PreVoted for a given block.
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
	return fmt.Sprintf("Polka(%s,Round=%d,Height=%d)", blockHeader, polka.Round, polka.Height)
}

// PolkaBuilder should start empty and be filled with valid
// SignedPreVote with Insert
type PolkaBuilder map[Height]map[sig.Signatory]SignedPreVote

func NewPolkaBuilder() PolkaBuilder {
	return PolkaBuilder{}
}

// Insert takes a valid SignedPreVote to register the vote. You can
// give this duplicate valid SignedPreVote and only one vote will be
// registered.
func (builder PolkaBuilder) Insert(preVote SignedPreVote) {
	if _, ok := builder[preVote.Height]; !ok {
		builder[preVote.Height] = map[sig.Signatory]SignedPreVote{}
	}
	if _, ok := builder[preVote.Height][preVote.Signatory]; !ok {
		builder[preVote.Height][preVote.Signatory] = preVote
	}
}

// Polka finds the `Polka` with the highest height that has enough
// `PreVotes` to be greater than or equal to the `consensusThreshold`
//
// By construction duplicate votes from the same signatory will only
// count as one vote. However, it does assume that each SignedPreVote
// has a valid header and signature.
func (builder PolkaBuilder) Polka(consensusThreshold int) (Polka, bool) {
	highestPolkaFound := false
	highestPolka := Polka{}
	for height, preVotes := range builder {
		if !highestPolkaFound || height > highestPolka.Height {
			if len(preVotes) < consensusThreshold {
				continue
			}
			highestPolkaFound = true

			// Note: also not used
			preVotesForNil := 0

			preVotesForBlock := map[sig.Hash]int{}
			for _, preVote := range preVotes {
				if preVote.Block == nil {
					// This is dead code since `preVotesForNil` is
					// never used
					preVotesForNil++

					// This does let us skip nill blocks, but we
					// already need to check for valid headers and
					// signatures before any votes reach to this
					// function. In other words, I think we should
					// remove the whole if statement here.
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

			// I am unable to get this code to be run given the
			// assumption that only valid SignedPreVote will be
			// inserted. Let me know what case this is supposed to
			// catch and if we need it here.
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

// Drop removes all registered SignedPreVote below the given height.
// Essentially, call this on your current height when you have found a
// polka.
func (builder PolkaBuilder) Drop(dropHeight Height) {
	for height := range builder {
		if height < dropHeight {
			delete(builder, height)
		}
	}
}
