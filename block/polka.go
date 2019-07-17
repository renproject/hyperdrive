package block

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/renproject/hyperdrive/sig"
	"golang.org/x/crypto/sha3"
)

// PreVote is you voting for the encapsulated block to be added
// during this round at this height. It is a PreVoteNil when the block
// is nil.
type PreVote struct {
	Block  *SignedBlock
	Round  Round
	Height Height
}

// NewPreVote creates a PreVote
func NewPreVote(block *SignedBlock, round Round, height Height) PreVote {
	return PreVote{
		Block:  block,
		Round:  round,
		Height: height,
	}
}

// Sign is indented to be the way you sign your PreVote with your
// private key
func (preVote PreVote) Sign(signer sig.Signer) (SignedPreVote, error) {
	//FIXME: this does not guarantee the contents of the block
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
	return fmt.Sprintf("PreVote(Height=%d,Round=%d,%s)", preVote.Height, preVote.Round, block)
}

func (preVote PreVote) Write(w io.Writer) error {
	if preVote.Block == nil {
		if err := binary.Write(w, binary.LittleEndian, uint8(0)); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.LittleEndian, uint8(1)); err != nil {
			return err
		}
		if err := preVote.Block.Write(w); err != nil {
			return err
		}
	}
	if err := binary.Write(w, binary.LittleEndian, preVote.Round); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, preVote.Height); err != nil {
		return err
	}
	return nil
}

func (preVote *PreVote) Read(r io.Reader) error {
	var hasBlock uint8
	if err := binary.Read(r, binary.LittleEndian, &hasBlock); err != nil {
		return err
	}
	if hasBlock == 1 {
		preVote.Block = new(SignedBlock)
		if err := preVote.Block.Read(r); err != nil {
			return err
		}
	}
	if err := binary.Read(r, binary.LittleEndian, &preVote.Round); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &preVote.Height); err != nil {
		return err
	}
	return nil
}

// SignedPreVote is the signed version of a PreVote
type SignedPreVote struct {
	PreVote
	Signature sig.Signature
	Signatory sig.Signatory
}

func (signedPreVote SignedPreVote) String() string {
	return fmt.Sprintf("SignedPreVote(%s,Signature=%v,Signatory=%v)", signedPreVote.PreVote.String(), signedPreVote.Signature, signedPreVote.Signatory)
}

func (signedPreVote SignedPreVote) Write(w io.Writer) error {
	if err := signedPreVote.PreVote.Write(w); err != nil {
		return err
	}
	if err := signedPreVote.Signature.Write(w); err != nil {
		return err
	}
	if err := signedPreVote.Signatory.Write(w); err != nil {
		return err
	}
	return nil
}

func (signedPreVote *SignedPreVote) Read(r io.Reader) error {
	if err := signedPreVote.PreVote.Read(r); err != nil {
		return err
	}
	if err := signedPreVote.Signature.Read(r); err != nil {
		return err
	}
	if err := signedPreVote.Signatory.Read(r); err != nil {
		return err
	}
	return nil
}

// Polka is created when 2/3 of the nodes have all PreVoted for a given block.
// The Signatures are expected to be ordered to match the order of the Signatories.
type Polka struct {
	Block       *SignedBlock
	Round       Round
	Height      Height
	Signatures  sig.Signatures
	Signatories sig.Signatories
}

// Equal checks that two Polka are functionally equivalent
func (polka Polka) Equal(other *Polka) bool {
	if polka.Block == nil && other.Block == nil {
		return polka.Round == other.Round &&
			polka.Height == other.Height &&
			polka.Signatures.Equal(other.Signatures) &&
			polka.Signatories.Equal(other.Signatories)
	}
	if polka.Block == nil || other.Block == nil {
		return false
	}
	return polka.Block.Equal(other.Block.Block) &&
		polka.Round == other.Round &&
		polka.Height == other.Height &&
		polka.Signatures.Equal(other.Signatures) &&
		polka.Signatories.Equal(other.Signatories)
}

func (polka Polka) String() string {
	blockHeader := "Nil"
	if polka.Block != nil {
		blockHeader = base64.StdEncoding.EncodeToString(polka.Block.Header[:])
	}
	return fmt.Sprintf("Polka(Height=%d,Round=%d,BlockHeader=%s)", polka.Height, polka.Round, blockHeader)
}

func (polka Polka) Write(w io.Writer) error {
	if polka.Block == nil {
		if err := binary.Write(w, binary.LittleEndian, uint8(0)); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.LittleEndian, uint8(1)); err != nil {
			return err
		}
		if err := polka.Block.Write(w); err != nil {
			return err
		}
	}
	if err := binary.Write(w, binary.LittleEndian, polka.Round); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, polka.Height); err != nil {
		return err
	}
	if err := polka.Signatures.Write(w); err != nil {
		return err
	}
	if err := polka.Signatories.Write(w); err != nil {
		return err
	}
	return nil
}

func (polka *Polka) Read(r io.Reader) error {
	var hasBlock uint8
	if err := binary.Read(r, binary.LittleEndian, &hasBlock); err != nil {
		return err
	}
	if hasBlock == 1 {
		polka.Block = new(SignedBlock)
		if err := polka.Block.Read(r); err != nil {
			return err
		}
	}
	if err := binary.Read(r, binary.LittleEndian, &polka.Round); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &polka.Height); err != nil {
		return err
	}
	polka.Signatures = sig.Signatures{}
	if err := polka.Signatures.Read(r); err != nil {
		return err
	}
	polka.Signatories = sig.Signatories{}
	if err := polka.Signatories.Read(r); err != nil {
		return err
	}
	return nil
}

// PolkaBuilder is used to build up collections of SignedPreVotes at different Heights and Rounds and then build Polkas
// wherever there are enough SignedPreVotes to do so.
type PolkaBuilder map[Height]map[Round]map[sig.Signatory]SignedPreVote

func NewPolkaBuilder() PolkaBuilder {
	return PolkaBuilder{}
}

// Insert a SignedPreVote into the PolkaBuilder. This will include the SignedPreVote in all attempts to build a Polka
// for the respective Height.
func (builder PolkaBuilder) Insert(preVote SignedPreVote) bool {
	// Pre-condition check
	if preVote.Block != nil {
		if preVote.Block.Height != preVote.Height {
			panic(fmt.Errorf("expected pre-vote height (%v) to equal pre-vote block height (%v)", preVote.Height, preVote.Block.Height))
		}
	}

	if _, ok := builder[preVote.Height]; !ok {
		builder[preVote.Height] = map[Round]map[sig.Signatory]SignedPreVote{}
	}
	if _, ok := builder[preVote.Height][preVote.Round]; !ok {
		builder[preVote.Height][preVote.Round] = map[sig.Signatory]SignedPreVote{}
	}
	if _, ok := builder[preVote.Height][preVote.Round][preVote.Signatory]; !ok {
		builder[preVote.Height][preVote.Round][preVote.Signatory] = preVote
		return true
	}
	return false
}

// Polka returns a Polka and the latest Round for which there are 2/3rd+
// pre-votes. The Polka will be nil unless there is 2/3rds+ pre-votes for nil or
// a specific SignedBlock. The Round will be nil unless there are 2/3rds+
// pre-votes for a specific Round. If a Polka is returned, the Round will match
// the Polka.
func (builder PolkaBuilder) Polka(height Height, consensusThreshold int) (*Polka, *Round) {
	// Pre-condition check
	if consensusThreshold < 1 {
		panic(fmt.Errorf("expected consensus threshold (%v) to be greater than 0", consensusThreshold))
	}

	// Short-circuit when too few pre-votes have been received
	preVotesByRound, ok := builder[height]
	if !ok {
		return nil, nil
	}

	var polka *Polka
	var preVotesRound *Round

	for round, preVotes := range preVotesByRound {

		numNilPreVotes := 0
		nilPreVoteSignatures := []sig.Signature{}
		nilPreVoteSignatories := []sig.Signatory{}

		if polka != nil && round <= polka.Round {
			continue
		}
		if len(preVotes) < consensusThreshold {
			continue
		}
		preVotesRound = &round

		// Build a mapping of the pre-votes for each block
		preVotesForBlock := map[sig.Hash]int{}
		for _, preVote := range preVotes {
			// Invariant check
			if preVote.Height != height {
				panic(fmt.Errorf("expected pre-vote height (%v) to equal %v", preVote.Height, height))
			}
			if preVote.Round != round {
				panic(fmt.Errorf("expected pre-vote round (%v) to equal %v", preVote.Round, round))
			}
			if preVote.Block == nil {
				numNilPreVotes++
				nilPreVoteSignatures = append(nilPreVoteSignatures, preVote.Signature)
				nilPreVoteSignatories = append(nilPreVoteSignatories, preVote.Signatory)
				continue
			}

			// Invariant check
			if preVote.Block.Height != height {
				panic(fmt.Errorf("expected pre-vote block height (%v) to equal %v", preVote.Block.Height, height))
			}
			if preVote.Round != round {
				panic(fmt.Errorf("expected pre-vote round (%v) to equal %v", preVote.Round, round))
			}
			preVotesForBlock[preVote.Block.Header]++
		}

		// Search for a polka of pre-votes for non-nil block
		for blockHeader, numPreVotes := range preVotesForBlock {
			if numPreVotes >= consensusThreshold {
				polka = &Polka{
					Signatures:  make(sig.Signatures, 0, consensusThreshold),
					Signatories: make(sig.Signatories, 0, consensusThreshold),
				}
				for _, preVote := range preVotes {
					if preVote.Block != nil && preVote.Block.Header.Equal(blockHeader) {
						if polka.Block != nil {
							// Invariant check
							if polka.Round != preVote.Round {
								panic(fmt.Errorf("expected polka round (%v) to equal pre-vote round (%v)", polka.Round, preVote.Round))
							}
							if polka.Height != preVote.Height {
								panic(fmt.Errorf("expected polka height (%v) to equal pre-vote height (%v)", polka.Height, preVote.Height))
							}
						} else {
							// Invariant check
							if preVote.Height != height {
								panic(fmt.Errorf("expected pre-vote height (%v) to equal %v", preVote.Height, height))
							}
							if preVote.Round != round {
								panic(fmt.Errorf("expected pre-vote round (%v) to equal %v", preVote.Round, round))
							}
							polka.Block = preVote.Block
							polka.Round = preVote.Round
							polka.Height = preVote.Height
						}
						polka.Signatures = append(polka.Signatures, preVote.Signature)
						polka.Signatories = append(polka.Signatories, preVote.Signatory)
					}
				}
				break
			}
		}

		if numNilPreVotes >= consensusThreshold {
			// Return a nil-Polka
			polka = &Polka{
				Block:       nil,
				Height:      height,
				Round:       round,
				Signatures:  nilPreVoteSignatures,
				Signatories: nilPreVoteSignatories,
			}
		}
	}

	if polka != nil {
		// Post-condition check
		if polka.Block != nil {
			if len(polka.Signatures) != len(polka.Signatories) {
				panic(fmt.Errorf("expected the number of signatures (%v) to be equal to the number of signatories (%v)", len(polka.Signatures), len(polka.Signatories)))
			}
			if len(polka.Signatures) < consensusThreshold {
				panic(fmt.Errorf("expected the number of signatures (%v) to be greater than or equal to the consensus threshold (%v)", len(polka.Signatures), consensusThreshold))
			}
		}
		if polka.Height != height {
			panic(fmt.Errorf("expected the polka height (%v) to equal %v", polka.Height, height))
		}
		return polka, &polka.Round
	}
	return nil, preVotesRound
}

// Drop removes all SignedPreVotes below the given Height.
func (builder PolkaBuilder) Drop(dropHeight Height) {
	for height := range builder {
		if height < dropHeight {
			delete(builder, height)
		}
	}
}
