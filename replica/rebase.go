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
	// NextBlock returns the `block.Txs`, `block.Plan` and the parent
	// `block.State` for the given `block.Height`.
	NextBlock(block.Kind, block.Height, Shard) (block.Txs, block.Plan, block.State)
}

type Validator interface {
	IsBlockValid(block block.Block, checkHistory bool, shard Shard) (process.NilReasons, error)
}

type Observer interface {
	DidCommitBlock(block.Height, Shard)
	DidReceiveSufficientNilPrevotes(messages process.Messages, f int)
	IsSignatory(Shard) bool
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

	txs, plan, prevState := rebaser.blockIterator.NextBlock(
		rebaser.expectedKind,
		height,
		rebaser.shard,
	)

	header := block.NewHeader(
		rebaser.expectedKind,
		parent.Hash(),
		base.Hash(),
		txs.Hash(),
		plan.Hash(),
		prevState.Hash(),
		height,
		round,
		block.Timestamp(time.Now().Unix()),
		expectedSigs,
	)

	return block.New(header, txs, plan, prevState)
}

func (rebaser *shardRebaser) IsBlockValid(proposedBlock block.Block, checkHistory bool) (process.NilReasons, error) {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	nilReasons := make(process.NilReasons)

	// Check the expected `block.Kind`
	if proposedBlock.Header().Kind() != rebaser.expectedKind {
		return nilReasons, fmt.Errorf("unexpected block kind: expected %v, got %v", rebaser.expectedKind, proposedBlock.Header().Kind())
	}
	switch proposedBlock.Header().Kind() {
	case block.Standard:
		if proposedBlock.Header().Signatories() != nil && len(proposedBlock.Header().Signatories()) != 0 {
			return nilReasons, fmt.Errorf("expected standard block to have nil/empty signatories")
		}

	case block.Rebase:
		if !proposedBlock.Header().Signatories().Equal(rebaser.expectedRebaseSigs) {
			return nilReasons, fmt.Errorf("unexpected signatories in rebase block: expected %d, got %d", len(rebaser.expectedRebaseSigs), len(proposedBlock.Header().Signatories()))
		}
		// TODO: Transactions are expected to be nil (the plan is not expected
		// to be nil, because there are "default" computations that might need
		// to be done every block).

	case block.Base:
		if !proposedBlock.Header().Signatories().Equal(rebaser.expectedRebaseSigs) {
			return nilReasons, fmt.Errorf("unexpected signatories in base block: expected %d, got %d", len(rebaser.expectedRebaseSigs), len(proposedBlock.Header().Signatories()))
		}
		if proposedBlock.Txs() != nil && len(proposedBlock.Txs()) != 0 {
			return nilReasons, fmt.Errorf("expected base block to have nil/empty txs")
		}
		if proposedBlock.Plan() != nil && len(proposedBlock.Plan()) != 0 {
			return nilReasons, fmt.Errorf("expected base block to have nil/empty plan")
		}

	default:
		panic(fmt.Errorf("invariant violation: must not propose block kind=%v", rebaser.expectedKind))
	}

	// Check the expected `block.Hash`
	if !proposedBlock.Hash().Equal(block.ComputeHash(proposedBlock.Header(), proposedBlock.Txs(), proposedBlock.Plan(), proposedBlock.PreviousState())) {
		return nilReasons, fmt.Errorf("unexpected block hash for proposed block")
	}

	// Check against the parent `block.Block`
	if checkHistory {
		parentBlock, ok := rebaser.blockStorage.Blockchain(rebaser.shard).BlockAtHeight(proposedBlock.Header().Height() - 1)
		if !ok {
			return nilReasons, fmt.Errorf("block at height=%d not found", proposedBlock.Header().Height()-1)
		}
		if proposedBlock.Header().Timestamp() < parentBlock.Header().Timestamp() {
			return nilReasons, fmt.Errorf("expected timestamp for proposed block to be greater than parent block")
		}
		if proposedBlock.Header().Timestamp() > block.Timestamp(time.Now().Unix()) {
			return nilReasons, fmt.Errorf("expected timestamp for proposed block to be less than current time")
		}
		if !proposedBlock.Header().ParentHash().Equal(parentBlock.Hash()) {
			return nilReasons, fmt.Errorf("expected parent hash for proposed block to equal parent block hash")
		}

		// Check that the parent is the most recently finalised
		latestBlock := rebaser.blockStorage.LatestBlock(rebaser.shard)
		if !parentBlock.Hash().Equal(latestBlock.Hash()) {
			return nilReasons, fmt.Errorf("expected parent block hash to equal latest block hash")
		}
		if parentBlock.Hash().Equal(block.InvalidHash) {
			return nilReasons, fmt.Errorf("parent block hash should not be invalid")
		}
	}

	// Check against the base `block.Block`
	baseBlock := rebaser.blockStorage.LatestBaseBlock(rebaser.shard)
	if !proposedBlock.Header().BaseHash().Equal(baseBlock.Hash()) {
		return nilReasons, fmt.Errorf("expected base hash for proposed block to equal base block hash")
	}

	// Pass to the next `process.Validator`
	if rebaser.validator != nil {
		return rebaser.validator.IsBlockValid(proposedBlock, checkHistory, rebaser.shard)
	}
	return nilReasons, nil
}

func (rebaser *shardRebaser) DidCommitBlock(height block.Height) {
	rebaser.mu.Lock()
	defer rebaser.mu.Unlock()

	committedBlock, ok := rebaser.blockStorage.Blockchain(rebaser.shard).BlockAtHeight(height)
	if !ok {
		panic(fmt.Errorf("invariant violation: missing block at height=%v", height))
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
		rebaser.observer.DidCommitBlock(height, rebaser.shard)
	}
}

func (rebaser *shardRebaser) DidReceiveSufficientNilPrevotes(messages process.Messages, f int) {
	if rebaser.observer != nil {
		rebaser.observer.DidReceiveSufficientNilPrevotes(messages, f)
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
