package replica

import (
	"fmt"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

// BlockStorage extends the `process.Blockchain` interface with the
// functionality to load the last committed `block.Standard`, and the last
// committed `block.Base`.
type BlockStorage interface {
	Blockchain(shard Shard) process.Blockchain
	LatestBlock(shard Shard) block.Block
	LatestBaseBlock(shard Shard) block.Block
}

type BlockIterator interface {
	// NextBlock returns the `block.Data` and the parent `block.State` for the
	// given `block.Height`.
	NextBlock(block.Kind, block.Height, Shard) (block.Data, block.State)
}

type Validator interface {
	IsBlockValid(block block.Block, checkHistory bool, shard Shard) bool
}

type Observer interface {
	DidCommitBlock(block.Height, Shard)
	IsSignatory() bool
}

type shardRebaser struct {
	mu *sync.Mutex

	expectedKind       block.Kind
	expectedRebaseSigs id.Signatories

	blockStorage  BlockStorage
	blockIterator BlockIterator
	validator     Validator
	observer      Observer
	shard         Shard
}

func newShardRebaser(blockStorage BlockStorage, blockIterator BlockIterator, validator Validator, observer Observer, shard Shard) *shardRebaser {
	return &shardRebaser{
		mu: new(sync.Mutex),

		expectedKind:       block.Standard,
		expectedRebaseSigs: nil,

		blockStorage:  blockStorage,
		blockIterator: blockIterator,
		validator:     validator,
		observer:      observer,
		shard:         shard,
	}
}

func (rebaser *shardRebaser) BlockProposal(height block.Height, round block.Round) block.Block {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	parent := rebaser.blockStorage.LatestBlock(rebaser.shard)
	base := rebaser.blockStorage.LatestBaseBlock(rebaser.shard)

	// Check that the base `block.Block` is a valid
	if base.Header().Kind() != block.Base {
		panic(fmt.Errorf("invariant violation: latest base block=%v has unexpected kind=%v", base.Hash(), base.Header().Kind()))
	}
	if base.Header().Signatories() == nil || len(base.Header().Signatories()) == 0 {
		panic(fmt.Errorf("invariant violation: latest base block=%v has unexpected empty signatories", base.Hash()))
	}

	var expectedSigs id.Signatories

	switch rebaser.expectedKind {
	case block.Standard:
		// Standard `block.Blocks` must not propose any `id.Signatories` in
		// their `block.Header`
		expectedSigs = nil
	case block.Rebase, block.Base:
		// Rebase/base `block.Blocks` must propose new `id.Signatories` in their
		// `block.Header`
		expectedSigs = make(id.Signatories, len(rebaser.expectedRebaseSigs))
		copy(expectedSigs, rebaser.expectedRebaseSigs)
	default:
		panic(fmt.Errorf("invariant violation: must not propose block kind=%v", rebaser.expectedKind))
	}

	header := block.NewHeader(
		rebaser.expectedKind,
		parent.Hash(),
		base.Hash(),
		height,
		round,
		block.Timestamp(time.Now().Unix()),
		expectedSigs,
	)
	data, prevState := rebaser.blockIterator.NextBlock(
		rebaser.expectedKind,
		height,
		rebaser.shard,
	)

	return block.New(header, data, prevState)
}

func (rebaser *shardRebaser) IsBlockValid(proposedBlock block.Block, checkHistory bool) bool {
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
	if !proposedBlock.Hash().Equal(block.ComputeHash(proposedBlock.Header(), proposedBlock.Data(), proposedBlock.PreviousState())) {
		return false
	}

	// Check against the parent `block.Block`
	if checkHistory {
		parentBlock, ok := rebaser.blockStorage.Blockchain(rebaser.shard).BlockAtHeight(proposedBlock.Header().Height() - 1)
		if !ok {
			return false
		}
		if proposedBlock.Header().Timestamp() < parentBlock.Header().Timestamp() {
			return false
		}
		if proposedBlock.Header().Timestamp() > block.Timestamp(time.Now().Unix()) {
			return false
		}
		if !proposedBlock.Header().ParentHash().Equal(parentBlock.Hash()) {
			return false
		}

		// Check that the parent is the most recently finalised
		latestBlock := rebaser.blockStorage.LatestBlock(rebaser.shard)
		if !parentBlock.Hash().Equal(latestBlock.Hash()) {
			return false
		}
		if parentBlock.Hash().Equal(block.InvalidHash) {
			return false
		}
	}

	// Check against the base `block.Block`
	// TODO : THIS NEEDS TO BE FIXED
	baseBlock := rebaser.blockStorage.LatestBaseBlock(rebaser.shard)
	if !proposedBlock.Header().BaseHash().Equal(baseBlock.Hash()) {
		return false
	}

	// Pass to the next `process.Validator`
	if rebaser.validator != nil {
		return rebaser.validator.IsBlockValid(proposedBlock, checkHistory, rebaser.shard)
	}
	return true
}

func (rebaser *shardRebaser) DidCommitBlock(height block.Height) {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	committedBlock, ok := rebaser.blockStorage.Blockchain(rebaser.shard).BlockAtHeight(height)
	if !ok {
		panic(fmt.Errorf("invariant violatoin: missing block at height=%v", height))
	}

	switch committedBlock.Header().Kind() {
	case block.Standard:
	case block.Rebase:
		rebaser.expectedKind = block.Base
		rebaser.expectedRebaseSigs = committedBlock.Header().Signatories() // TODO: Should this be an invariant check?
	case block.Base:
		rebaser.expectedKind = block.Standard
		rebaser.expectedRebaseSigs = nil
	}
	if rebaser.observer != nil {
		// TODO : external observer needs to take the previous state in the committedBlock and store in the storage
		rebaser.observer.DidCommitBlock(height, rebaser.shard)
	}
}

func (rebaser *shardRebaser) rebase(sigs id.Signatories) {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	if rebaser.expectedKind != block.Standard {
		// Handle duplicate rebase calls
		if !sigs.Equal(rebaser.expectedRebaseSigs) {
			panic("invariant violation: must not rebase while rebasing")
		}
		return
	}

	rebaser.expectedKind = block.Rebase
	rebaser.expectedRebaseSigs = sigs
}
