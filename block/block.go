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

func (block Block) String() string {
	return fmt.Sprintf("Block(Header=%s,Round=%d,Height=%d)", base64.StdEncoding.EncodeToString(block.Header[:]), block.Round, block.Height)
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

func (blockchain *Blockchain) Head() (Block, bool) {
	if blockchain.head.Polka.Block == nil {
		return Genesis(), false
	}
	return *blockchain.head.Polka.Block, true
}

func (blockchain *Blockchain) Block(header sig.Hash) (Block, bool) {
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
