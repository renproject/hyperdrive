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
		txHeader := tx.Header()
		copy(txHeaders[32*i:], txHeader[:])
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
	*SignedBlock
	Round Round
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
	return fmt.Sprintf("Propose(%s)", propose.Block.String())
}

type SignedPropose struct {
	Propose

	Signature sig.Signature
	Signatory sig.Signatory
}

type Blockchain struct {
	head   Commit
	blocks map[Height]Commit
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
		blocks: map[Height]Commit{genesis.Height: genesisCommit},
	}
}

func (blockchain *Blockchain) Height() Height {
	if blockchain.head.Polka.Block == nil {
		return Genesis().Height
	}
	return blockchain.head.Polka.Block.Height
}

func (blockchain *Blockchain) Round() *Round {
	if blockchain.head.Polka.Block == nil {
		return nil
	}
	return &blockchain.head.Polka.Round
}

func (blockchain *Blockchain) Head() (SignedBlock, bool) {
	if blockchain.head.Polka.Block == nil {
		return Genesis(), false
	}
	return *blockchain.head.Polka.Block, true
}

func (blockchain *Blockchain) Block(height Height) (SignedBlock, bool) {
	commit, ok := blockchain.blocks[height]
	if !ok || commit.Polka.Block == nil {
		return Genesis(), false
	}
	return *commit.Polka.Block, true
}

func (blockchain *Blockchain) Extend(commitToNextBlock Commit) {
	if commitToNextBlock.Polka.Block == nil {
		return
	}
	blockchain.blocks[commitToNextBlock.Polka.Block.Height] = commitToNextBlock
	if blockchain.Height() < commitToNextBlock.Polka.Block.Height {
		blockchain.head = commitToNextBlock
	}
}

func (blockchain *Blockchain) Blocks(blockNumber Height, n int64) []Commit {
	var block Commit
	var ok bool

	blocks := []Commit{}
	for i := blockNumber; i < blockNumber+Height(n); i++ {
		if block, ok = blockchain.blocks[i]; !ok {
			return blocks
		}
		blocks = append(blocks, block)
	}
	return blocks
}
