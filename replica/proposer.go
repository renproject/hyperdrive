package replica

import (
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/block"
)

type BlockStorage interface {
	block.Blockchain

	LatestBlock() block.Block
	LatestBaseBlock() block.Block
}

type BlockDataIterator interface {
	Next() block.Data
}

func BlockProposer(blockStorage BlockStorage, blockDataIterator BlockDataIterator) process.Proposer {
	return &blockProposer{
		blockStorage:      blockStorage,
		blockDataIterator: blockDataIterator,
	}
}

type blockProposer struct {
	blockStorage      BlockStorage
	blockDataIterator BlockDataIterator
}

func (proposer *blockProposer) Propose(height block.Height, round block.Round) block.Block {
	parent := proposer.blockStorage.LatestBlock()
	base := proposer.blockStorage.LatestBaseBlock()
	if base.Header().Kind() != block.Base {
		panic(fmt.Errorf("invariant violation: latest base block=%v has unexpected kind=%v", base.Hash(), base.Header().Kind()))
	}
	if base.Header().Signatories() == nil {
		panic(fmt.Errorf("invariant violation: latest base block=%v has unexpected nil signatories", base.Hash()))
	}
	header := block.NewHeader(
		block.Standard,
		parent.Hash(),
		base.Hash(),
		height,
		round,
		block.Timestamp(time.Now().Unix()),
		base.Header().Signatories(),
	)
	content := proposer.blockDataIterator.Next()
	return block.New(header, content)
}
