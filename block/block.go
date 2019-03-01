package block

import (
	"time"

	"github.com/renproject/hyperdrive/sig"
)

type Round int64

type Height int64

type Block struct {
	Time         time.Time
	Round        Round
	Height       Height
	Header       sig.Hash
	ParentHeader sig.Hash
	Signature    sig.Signature
	Signatory    sig.Signatory
}

type Blockchain struct {
	head Block
	tail map[sig.Hash]Block
}

func (blockchain *Blockchain) Height() Height {
	return blockchain.head.Height
}

func (blockchain *Blockchain) Round() Round {
	return blockchain.head.Round
}

func (blockchain *Blockchain) Head() Block {
	return blockchain.head
}

func (blockchain *Blockchain) Block(header sig.Hash) (Block, bool) {
	block, ok := blockchain.tail[header]
	return block, ok
}

func (blockchain *Blockchain) Extend(nextBlock Block) {
	blockchain.tail[nextBlock.Header] = nextBlock
	blockchain.head = nextBlock
}
