package replica

import (
	"fmt"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
)

// BlockStorage extends the `process.Blockchain` interface with the
// functionality to load the last committed `block.Standard`, and the last
// committed `block.Base`.
type BlockStorage interface {
	block.Blockchain

	LatestBlock() block.Block
	LatestBaseBlock() block.Block
}

type BlockDataIterator interface {
	Next(block.Kind) block.Data
}

type ShardRebaser interface {
	process.Proposer
	process.Validator
	process.Observer

	Rebase(toSigs block.Signatories)
}

func NewShardRebaser(blockStorage BlockStorage, blockDataIterator BlockDataIterator, validator process.Validator, observer process.Observer) ShardRebaser {
	return &shardRebaser{
		mu: new(sync.Mutex),

		expectedKind:       block.Standard,
		expectedRebaseSigs: nil,

		blockStorage:      blockStorage,
		blockDataIterator: blockDataIterator,
		validator:         validator,
		observer:          observer,
	}
}

type shardRebaser struct {
	mu *sync.Mutex

	expectedKind       block.Kind
	expectedRebaseSigs block.Signatories

	blockStorage      BlockStorage
	blockDataIterator BlockDataIterator
	validator         process.Validator
	observer          process.Observer
}

func (rebaser *shardRebaser) Rebase(sigs block.Signatories) {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	if rebaser.expectedKind != block.Standard {
		if !sigs.Equal(rebaser.expectedRebaseSigs) {
			panic("invariant violation: must not rebase while rebasing")
		}
		return
	}

	rebaser.expectedKind = block.Rebase
	rebaser.expectedRebaseSigs = sigs
}

func (rebaser *shardRebaser) Propose(height block.Height, round block.Round) block.Block {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	parent := rebaser.blockStorage.LatestBlock()
	base := rebaser.blockStorage.LatestBaseBlock()

	// Check that the base `block.Block` is a valid
	if base.Header().Kind() != block.Base {
		panic(fmt.Errorf("invariant violation: latest base block=%v has unexpected kind=%v", base.Hash(), base.Header().Kind()))
	}
	if base.Header().Signatories() == nil {
		panic(fmt.Errorf("invariant violation: latest base block=%v has unexpected nil signatories", base.Hash()))
	}

	var header block.Header
	var data block.Data

	switch rebaser.expectedKind {
	case block.Standard:
		// Propose a standard `block.Block`
		header = block.NewHeader(
			rebaser.expectedKind,
			parent.Hash(),
			base.Hash(),
			height,
			round,
			block.Timestamp(time.Now().Unix()),
			base.Header().Signatories(),
		)
		data = rebaser.blockDataIterator.Next(
			rebaser.expectedKind,
		)

	case block.Rebase:
		// Propose a rebase `block.Block` with the expected rebase
		// `block.Signatories`
		header = block.NewHeader(
			rebaser.expectedKind,
			parent.Hash(),
			base.Hash(),
			height,
			round,
			block.Timestamp(time.Now().Unix()),
			rebaser.expectedRebaseSigs,
		)
		data = rebaser.blockDataIterator.Next(
			rebaser.expectedKind,
		)

	case block.Base:
		// Propose a base `block.Block` with nil `block.Data` the expected
		// rebase `block.Signatories`
		header = block.NewHeader(
			rebaser.expectedKind,
			parent.Hash(),
			base.Hash(),
			height,
			round,
			block.Timestamp(time.Now().Unix()),
			rebaser.expectedRebaseSigs,
		)

	default:
		panic(fmt.Errorf("invariant violation: must not propose block kind=%v", rebaser.expectedKind))
	}

	return block.New(header, data)
}

func (rebaser *shardRebaser) Validate(proposedBlock block.Block) bool {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	// Check the expected `block.Kind`
	if proposedBlock.Header().Kind() != rebaser.expectedKind {
		return false
	}
	switch proposedBlock.Header().Kind() {
	case block.Standard:
		if proposedBlock.Header().Signatories() != nil {
			return false
		}

	case block.Rebase:
		if !proposedBlock.Header().Signatories().Equal(rebaser.expectedRebaseSigs) {
			return false
		}

	case block.Base:
		if !proposedBlock.Header().Signatories().Equal(rebaser.expectedRebaseSigs) {
			return false
		}
		if proposedBlock.Data() != nil {
			return false
		}

	default:
		panic(fmt.Errorf("invariant violation: must not propose block kind=%v", rebaser.expectedKind))
	}

	// Check the expected `block.Hash`
	if !proposedBlock.Hash().Equal(block.NewHash(proposedBlock.Header(), proposedBlock.Data())) {
		return false
	}

	// Check against the parent `block.Block`
	parentBlock, ok := rebaser.blockStorage.BlockAtHeight(proposedBlock.Header().Height() - 1)
	if !ok {
		return false
	}
	if proposedBlock.Header().Timestamp() > parentBlock.Header().Timestamp() {
		return false
	}
	if proposedBlock.Header().Timestamp() < block.Timestamp(time.Now().Unix()) {
		return false
	}
	if !proposedBlock.Header().ParentHash().Equal(parentBlock.Note().Hash()) {
		return false
	}

	// Check against the base `block.Block`
	baseBlock := rebaser.blockStorage.LatestBaseBlock()
	if !proposedBlock.Header().BaseHash().Equal(baseBlock.Note().Hash()) {
		return false
	}

	// Check that the parent is the most recently finalised
	latestBlock := rebaser.blockStorage.LatestBlock()
	if !parentBlock.Note().Hash().Equal(latestBlock.Note().Hash()) {
		return false
	}
	if parentBlock.Note().Hash().Equal(block.InvalidHash) {
		return false
	}

	// Pass to the next `process.Validator`
	if rebaser.validator != nil {
		return rebaser.validator.Validate(proposedBlock)
	}
	return true
}

func (rebaser *shardRebaser) OnBlockCommitted(height block.Height) {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	committedBlock, ok := rebaser.blockStorage.BlockAtHeight(height)
	if !ok {
		panic(fmt.Errorf("invariant violatoin: missing block at height=%v", height))
	}

	switch committedBlock.Header().Kind() {
	case block.Standard:
	case block.Rebase:
		rebaser.expectedKind = block.Base
		rebaser.expectedRebaseSigs = committedBlock.Header().Signatories()
	case block.Base:
		rebaser.expectedKind = block.Standard
		rebaser.expectedRebaseSigs = nil
	}

	if rebaser.observer != nil {
		rebaser.observer.OnBlockCommitted(height)
	}
}
