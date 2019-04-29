package block

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
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
	Txs          tx.Transactions
}

func New(round Round, height Height, parentHeader sig.Hash, txs tx.Transactions) Block {
	block := Block{
		Time:         time.Now(),
		Round:        round,
		Height:       height,
		ParentHeader: parentHeader,
		Txs:          txs,
	}
	hashSum256 := calculateHeader(block)
	copy(block.Header[:], hashSum256[:])
	return block
}

// Equal excludes time from equality check
func (block Block) Equal(other Block) bool {
	return block.Round == other.Round &&
		block.Height == other.Height &&
		block.Header.Equal(other.Header) &&
		block.ParentHeader.Equal(other.ParentHeader)
}

func (block Block) Sign(signer sig.Signer) (SignedBlock, error) {
	signedBlock := SignedBlock{
		Block: block,
	}

	signature, err := signer.Sign(signedBlock.Header)
	if err != nil {
		return SignedBlock{}, err
	}
	signedBlock.Signature = signature
	signedBlock.Signatory = signer.Signatory()

	return signedBlock, nil
}

func (block Block) String() string {
	return fmt.Sprintf("Block(Header=%s,Round=%d,Height=%d)", base64.StdEncoding.EncodeToString(block.Header[:]), block.Round, block.Height)
}

type SignedBlock struct {
	Block

	Signature sig.Signature
	Signatory sig.Signatory
}

func Genesis() SignedBlock {
	return SignedBlock{
		Block: Block{
			Time:         time.Unix(0, 0),
			Round:        0,
			Height:       0,
			Header:       sig.Hash{},
			ParentHeader: sig.Hash{},
			Txs:          tx.Transactions{},
		},
		Signature: sig.Signature{},
		Signatory: sig.Signatory{},
	}
}

func (signedBlock SignedBlock) String() string {
	return fmt.Sprintf("Block(Header=%s,Round=%d,Height=%d)", base64.StdEncoding.EncodeToString(signedBlock.Header[:]), signedBlock.Round, signedBlock.Height)
}

type Blockchain struct {
	head   Commit
	blocks map[sig.Hash]Commit
}

func NewBlockchain() Blockchain {
	genesis := Genesis()
	genesisCommit := Commit{
		Polka: Polka{
			Block: &genesis,
		},
	}
	return Blockchain{
		head:   genesisCommit,
		blocks: map[sig.Hash]Commit{genesis.Header: genesisCommit},
	}
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

func (blockchain *Blockchain) Head() (SignedBlock, bool) {
	if blockchain.head.Polka.Block == nil {
		return Genesis(), false
	}
	return *blockchain.head.Polka.Block, true
}

func (blockchain *Blockchain) Block(header sig.Hash) (SignedBlock, bool) {
	commit, ok := blockchain.blocks[header]
	if !ok || commit.Polka.Block == nil {
		return Genesis(), false
	}
	return *commit.Polka.Block, true
}

func (blockchain *Blockchain) Extend(commitToNextBlock Commit) {
	if commitToNextBlock.Polka.Block == nil {
		return
	}
	blockchain.blocks[commitToNextBlock.Polka.Block.Header] = commitToNextBlock
	blockchain.head = commitToNextBlock
}

// calculateHeader will return the SHA3 hash of the parentHeader, timestamp, round,
// height, and transactions.
// TODO: (Review) Should block header include timestamp as well, given that `Equal`
// ignores Time when checking for equality between 2 blocks. (See comment in `Equal`)
func calculateHeader(block Block) [32]byte {
	headerString := fmt.Sprintf("Block(ParentHeader=%s,Timestamp=%s,Round=%d,Height=%d,Transactions=[", base64.StdEncoding.EncodeToString(block.ParentHeader[:]), block.Time.String(), block.Round, block.Height)

	isFirstTx := true
	for _, tx := range block.Txs {
		data, err := tx.Marshal()
		if err != nil {
			// FIXME: handle this error
			fmt.Printf("[calculateHeader] error marshalling transaction: %v\n", err)
			continue
		}
		if isFirstTx {
			headerString = fmt.Sprintf("%s%s", headerString, base64.StdEncoding.EncodeToString(data))
			isFirstTx = false
			continue
		}
		headerString = fmt.Sprintf("%s,%s", headerString, base64.StdEncoding.EncodeToString(data))
	}

	headerString = fmt.Sprintf("%s])", headerString)
	return sha3.Sum256([]byte(headerString))
}
