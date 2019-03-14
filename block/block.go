package block

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"reflect"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"golang.org/x/crypto/sha3"
)

// The Round in which a `Block` was proposed.
type Round int64

// The Height at which a `Block` was proposed.
type Height int64

type Block struct {
	Time time.Time
	//TODO: request clarification on Round
	Round        Round
	Height       Height
	Header       sig.Hash
	ParentHeader sig.Hash
	Signature    sig.Signature
	Signatory    sig.Signatory
}

// GenerateBlock makes a random Block with a given height, but random
// ParentHeader and signed with a random public key. Should
// be a valid block.
func GenerateBlock(rand *rand.Rand, height Height) Block {
	round := Round(height)

	reflectHash, errBool := quick.Value(reflect.TypeOf(sig.Hash{}), rand)
	if !errBool {
		panic("Block.Generate: parentHeader type reflect failed")
	}
	parentHeader := reflectHash.Interface().(sig.Hash)

	//TODO: check if this is the correct way to generate the hash of a
	//block
	data := []byte(fmt.Sprintf("%d%d%v", round, height, parentHeader))
	hashSum256 := sha3.Sum256(data)
	header := sig.Hash{}
	copy(header[:], hashSum256[:])

	newSV, err := ecdsa.NewFromRandom()
	if err != nil {
		panic("Block.Generate: ecdsa failed")
	}

	signature, err := newSV.Sign(header)
	if err != nil {
		panic("Block.Generate: sign failed")
	}

	return Block{
		Time:         time.Unix(0, 0),
		Round:        round,
		Height:       height,
		Header:       header,
		ParentHeader: parentHeader,
		Signature:    signature,
		Signatory:    newSV.Signatory(),
	}
}

// Equal excludes time from equality check
func (block Block) Equal(other Block) bool {
	return block.Round == other.Round &&
		block.Height == other.Height &&
		block.Header.Equal(other.Header) &&
		block.ParentHeader.Equal(other.ParentHeader) &&
		block.Signature.Equal(other.Signature) &&
		block.Signatory.Equal(other.Signatory)
}

func Genesis() Block {
	return Block{
		Time:         time.Unix(0, 0),
		Round:        0,
		Height:       0,
		Header:       sig.Hash{},
		ParentHeader: sig.Hash{},
		Signature:    sig.Signature{},
		Signatory:    sig.Signatory{},
	}
}

func (block Block) String() string {
	return fmt.Sprintf("Block(Header=%s,Round=%d,Height=%d)", base64.StdEncoding.EncodeToString(block.Header[:]), block.Round, block.Height)
}

type Blockchain struct {
	head Commit
	tail map[sig.Hash]Commit
}

func (blockchain *Blockchain) Height() Height {
	if blockchain.head.Polka.Block == nil {
		return Genesis().Height
	}
	return blockchain.head.Polka.Block.Height
}

func (blockchain *Blockchain) Round() Round {
	if blockchain.head.Polka.Block == nil {
		return Genesis().Round
	}
	return blockchain.head.Polka.Block.Round
}

func (blockchain *Blockchain) Head() (Block, bool) {
	if blockchain.head.Polka.Block == nil {
		return Genesis(), false
	}
	return *blockchain.head.Polka.Block, true
}

// Block finds the block for the given header
func (blockchain *Blockchain) Block(header sig.Hash) (Block, bool) {
	commit, ok := blockchain.tail[header]
	if !ok || commit.Polka.Block == nil {
		return Genesis(), false
	}
	return *commit.Polka.Block, true
}

func (blockchain *Blockchain) Extend(commitToNextBlock Commit) {
	if commitToNextBlock.Polka.Block == nil {
		return
	}
	blockchain.tail[commitToNextBlock.Polka.Block.Header] = commitToNextBlock
	blockchain.head = commitToNextBlock
}
