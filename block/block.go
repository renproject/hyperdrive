package block

import (
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
	"golang.org/x/crypto/sha3"
)

// MaxTransactions defines the maximum number of transactions allowed in a block.
const MaxTransactions = 8

// The Round in which a `Block` was proposed.
type Round int64

// The Height at which a `Block` was proposed.
type Height int64

type Block struct {
	Time         time.Time
	Height       Height
	Header       sig.Hash
	ParentHeader sig.Hash
	Txs          tx.Transactions
	TxHeader     sig.Hash
}

func New(height Height, parentHeader sig.Hash, txs tx.Transactions) Block {
	block := Block{
		Time:         time.Now(),
		Height:       height,
		ParentHeader: parentHeader,
		Txs:          txs,
	}
	txHeaders := make([]byte, 32*len(block.Txs))
	for i, tx := range block.Txs {
		copy(txHeaders[32*i:], tx[:])
	}
	block.TxHeader = sha3.Sum256(txHeaders)
	block.Header = sha3.Sum256([]byte(block.String()))
	return block
}

// Equal excludes time from equality check
func (block Block) Equal(other Block) bool {
	return block.Height == other.Height &&
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
	return fmt.Sprintf("Block(Height=%d,Timestamp=%d,TxHeader=%s,ParentHeader=%s)", block.Height, block.Time.Unix(), block.TxHeader, block.ParentHeader)
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
	return fmt.Sprintf("Block(Height=%d,Timestamp=%d,TxHeader=%s,ParentHeader=%s,Header=%s)", signedBlock.Height, signedBlock.Time.Unix(), signedBlock.TxHeader, signedBlock.ParentHeader, signedBlock.Header)
}

type Propose struct {
	Block      SignedBlock
	Round      Round
	ValidRound Round // TODO: (Review) This name comes from the pseudocode in (https://arxiv.org/pdf/1807.04938.pdf). Should this be renamed to something more appropriate?
	LastCommit *Commit
}

// Sign a Propose with your private key
func (propose Propose) Sign(signer sig.Signer) (SignedPropose, error) {
	data := []byte(propose.String())

	hashSum256 := sha3.Sum256(data)
	hash := sig.Hash{}
	copy(hash[:], hashSum256[:])

	signature, err := signer.Sign(hash)
	if err != nil {
		return SignedPropose{}, err
	}

	return SignedPropose{
		Propose:   propose,
		Signature: signature,
		Signatory: signer.Signatory(),
	}, nil
}

func (propose Propose) String() string {
	return fmt.Sprintf("Propose(Block=%s,Round=%d,ValidRound=%d)", propose.Block.String(), propose.Round, propose.ValidRound)
}

type SignedPropose struct {
	Propose

	Signature sig.Signature
	Signatory sig.Signatory
}

type Blockchain struct {
	blocks BlockStore
}

func NewBlockchain(blocks BlockStore) Blockchain {
	return Blockchain{
		blocks: blocks,
	}
}

func (blockchain *Blockchain) Height() Height {
	height, err := blockchain.blocks.Height()
	if err != nil {
		return Genesis().Height
	}
	return height
}

func (blockchain *Blockchain) Head() (Commit, bool) {
	head, err := blockchain.blocks.Head()
	if err != nil || head.Polka.Block == nil {
		genesis := Genesis()
		return Commit{Polka: Polka{Block: &genesis}}, false
	}
	return head, true
}

func (blockchain *Blockchain) Block(height Height) (Commit, bool) {
	commit, err := blockchain.blocks.Block(height)
	if err != nil || commit.Polka.Block == nil {
		genesis := Genesis()
		return Commit{Polka: Polka{Block: &genesis}}, false
	}
	return commit, true
}

func (blockchain *Blockchain) Extend(commitToNextBlock Commit) error {
	if commitToNextBlock.Polka.Block == nil {
		return nil
	}
	return blockchain.blocks.Extend(commitToNextBlock)
}

func (blockchain *Blockchain) Blocks(begin, end Height) []Commit {
	if end < begin {
		return []Commit{}
	}

	var block Commit
	var err error
	blocks := []Commit{}

	for i := begin; i <= end; i++ {
		if block, err = blockchain.blocks.Block(i); err != nil || block.Polka.Block == nil {
			return blocks
		}
		blocks = append(blocks, block)
	}
	return blocks
}

type BlockStore interface {
	// Extend inserts the commit to the BlockStore and updates its head
	Extend(commitToNextBlock Commit) error

	// Return block (if it exists) that corresponds to the given height
	Block(height Height) (Commit, error)

	// Head returns the last seen commit
	Head() (Commit, error)

	// Height returns the height of the last seen commit
	Height() (Height, error)
}
