package replica

import (
	"fmt"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/schedule"
	"github.com/renproject/id"
)

// BlockStorage extends the `process.Blockchain` interface with the
// functionality to load the last committed `block.Standard`, and the last
// committed `block.Base`.
type BlockStorage interface {
	Blockchain(shard process.Shard) process.Blockchain
	LatestBlock(shard process.Shard) block.Block
	LatestBaseBlock(shard process.Shard) block.Block
}

type BlockIterator interface {
	// NextBlock returns the `block.Txs`, `block.Plan` and the parent
	// `block.State` for the given `block.Height`.
	NextBlock(block.Kind, block.Height, process.Shard) (block.Txs, block.Plan, block.State)

	// MissedBaseBlocksInRange must return an upper bound estimate for the
	// number of missed base blocks between (exclusive) two blocks (identified
	// by their block hash). This is used to prevent forking by old signatories.
	// This function should not include the "end" block as a missed block if it
	// is a rebasing Propose as if this function is being called, it has clearly
	// not been missed. For example, if the range is heights 10 - 20 and there
	// was expected to be a base block at 18, then this function is expected to
	// return 1. However, if there was expected to be a block at height 20,
	// then this function would return 0.
	MissedBaseBlocksInRange(begin, end id.Hash) int
}

type Validator interface {
	IsBlockValid(block block.Block, checkHistory bool, shard process.Shard) (process.NilReasons, error)
}

type Observer interface {
	DidCommitBlock(block.Height, process.Shard)
	DidReceiveSufficientNilPrevotes(messages process.Messages, f int)
	IsSignatory(process.Shard) bool
}

type shardRebaser struct {
	mu *sync.Mutex

	expectedKind       block.Kind
	expectedRebaseSigs id.Signatories
	numSignatories     int

	scheduler     schedule.Scheduler
	blockStorage  BlockStorage
	blockIterator BlockIterator
	validator     Validator
	observer      Observer
	shard         process.Shard
}

func newShardRebaser(scheduler schedule.Scheduler, blockStorage BlockStorage, blockIterator BlockIterator, validator Validator, observer Observer, shard process.Shard, numSignatories int) *shardRebaser {
	return &shardRebaser{
		mu: new(sync.Mutex),

		expectedKind:       block.Standard,
		expectedRebaseSigs: nil,
		numSignatories:     numSignatories,

		scheduler:     scheduler,
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
	if checkHistory {
		if proposedBlock.Header().Kind() != rebaser.expectedKind {
			return nilReasons, fmt.Errorf("unexpected block kind: expected %v, got %v", rebaser.expectedKind, proposedBlock.Header().Kind())
		}
	}
	switch proposedBlock.Header().Kind() {
	case block.Standard:
		if proposedBlock.Header().Signatories() != nil && len(proposedBlock.Header().Signatories()) != 0 {
			return nilReasons, fmt.Errorf("expected standard block to have nil/empty signatories")
		}

	case block.Rebase:
		if checkHistory {
			if len(proposedBlock.Header().Signatories()) != rebaser.numSignatories {
				return nilReasons, fmt.Errorf("unexpected number of signatories in rebase block: expected %d, got %d", rebaser.numSignatories, len(proposedBlock.Header().Signatories()))
			}
			if !proposedBlock.Header().Signatories().Equal(rebaser.expectedRebaseSigs) {
				return nilReasons, fmt.Errorf("unexpected signatories in rebase block: expected %d, got %d", len(rebaser.expectedRebaseSigs), len(proposedBlock.Header().Signatories()))
			}
		}

	case block.Base:
		if checkHistory {
			if !proposedBlock.Header().Signatories().Equal(rebaser.expectedRebaseSigs) {
				return nilReasons, fmt.Errorf("unexpected signatories in base block: expected %d, got %d", len(rebaser.expectedRebaseSigs), len(proposedBlock.Header().Signatories()))
			}
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
	if !proposedBlock.Header().TxsRef().Equal(proposedBlock.Txs().Hash()) {
		if !(proposedBlock.Txs() == nil && proposedBlock.Header().TxsRef().Equal(id.Hash{})) {
			return nilReasons, fmt.Errorf("unexpected txs hash for proposed block")
		}
	}
	if !proposedBlock.Header().PlanRef().Equal(proposedBlock.Plan().Hash()) {
		if !(proposedBlock.Plan() == nil && proposedBlock.Header().PlanRef().Equal(id.Hash{})) {
			return nilReasons, fmt.Errorf("unexpected plan hash for proposed block")
		}
	}
	if !proposedBlock.Header().PrevStateRef().Equal(proposedBlock.PreviousState().Hash()) {
		if !(proposedBlock.PreviousState() == nil && proposedBlock.Header().PrevStateRef().Equal(id.Hash{})) {
			return nilReasons, fmt.Errorf("unexpected previous state hash for proposed block")
		}
	}
	if !proposedBlock.Hash().Equal(block.NewBlockHash(proposedBlock.Header(), proposedBlock.Txs(), proposedBlock.Plan(), proposedBlock.PreviousState())) {
		return nilReasons, fmt.Errorf("unexpected block hash for proposed block")
	}

	// Check against the parent `block.Block`
	if checkHistory {
		parentBlock, ok := rebaser.blockStorage.Blockchain(rebaser.shard).BlockAtHeight(proposedBlock.Header().Height() - 1)
		if !ok {
			return nilReasons, fmt.Errorf("block at height=%d not found", proposedBlock.Header().Height()-1)
		}
		if proposedBlock.Header().Timestamp() <= parentBlock.Header().Timestamp() {
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

	// Check against the base `block.Block`. Since it is assumed that base
	// blocks are not missed, this check happens regardless of the
	// `checkHistory` parameter.
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
		if rebaser.scheduler != nil {
			rebaser.scheduler.Rebase(committedBlock.Header().Signatories())
		}
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
