package block

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
)

// The Round in which a `Block` was proposed.
type Round int64

// The Height at which a `Block` was proposed.
type Height int64

type Block struct {
	Time         time.Time
	Round        Round
	Height       Height
	Header       sig.Hash
	ParentHeader sig.Hash
	Signature    sig.Signature
	Signatory    sig.Signatory
	Txs          tx.Transactions
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
		Txs:          tx.Transactions{},
	}
}

func New(round Round, height Height, parentHeader sig.Hash, txs tx.Transactions) Block {
	block := Block{
		Time:         time.Now(),
		Round:        round,
		Height:       height,
		ParentHeader: parentHeader,
		Txs:          txs,
	}
	// FIXME: At this point we should calculate the block header. It should be the SHA3 hash of the timestamp, round,
	// height, parentHeader, and transactions.
	block.Header = sig.Hash{}
	return block
}

func (block *Block) Sign(signer sig.Signer) (err error) {
	block.Signature, err = signer.Sign(block.Header)
	if err != nil {
		return
	}
	block.Signatory = signer.Signatory()
	return
}

func (block Block) String() string {
	return fmt.Sprintf("Block(Header=%s,Round=%d,Height=%d)", base64.StdEncoding.EncodeToString(block.Header[:]), block.Round, block.Height)
}

type Blockchain struct {
	head Commit
	tail map[sig.Hash]Commit
}

func NewBlockchain() Blockchain {
	genesis := Genesis()
	return Blockchain{
		head: Commit{
			Polka: Polka{
				Block: &genesis,
			},
		},
		tail: map[sig.Hash]Commit{},
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

func (blockchain *Blockchain) Head() (Block, bool) {
	if blockchain.head.Polka.Block == nil {
		return Genesis(), false
	}
	return *blockchain.head.Polka.Block, true
}

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
