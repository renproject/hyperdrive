package hyperdrive_test

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/state"
	"github.com/renproject/hyperdrive/testutils"
	"github.com/renproject/hyperdrive/tx"
	co "github.com/republicprotocol/co-go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive"
)

var _ = Describe("Hyperdrive", func() {

	initReplicas := func(n int) ([]chan Object, []sig.SignerVerifier, *time.Ticker, chan struct{}, int) {
		done := make(chan struct{})
		ipChans := make([]chan Object, n)
		signatories := make(sig.Signatories, n)
		signers := make([]sig.SignerVerifier, n)
		shardHash := testutils.RandomHash()

		txPool := tx.FIFOPool(100)
		go populateTxPool(txPool)

		for i := 0; i < n; i++ {
			var err error
			ipChans[i] = make(chan Object, n*n)
			signers[i], err = ecdsa.NewFromRandom()
			signatories[i] = signers[i].Signatory()
			Expect(err).ShouldNot(HaveOccurred())
		}

		shard := shard.Shard{
			Hash:        shardHash,
			Signatories: make(sig.Signatories, n),
		}
		copy(shard.Signatories[:], signatories[:])

		for i := 0; i < n; i++ {
			ipChans[i] <- ShardObject{shard, txPool}
		}

		tickerInterval := time.Duration(n * n * 2)
		if n <= 16 {
			tickerInterval = time.Duration(1000)
		}
		ticker := time.NewTicker(tickerInterval * time.Millisecond)
		go func() {
			for t := range ticker.C {
				for i := 0; i < n; i++ {
					select {
					case <-done:
						return
					case ipChans[i] <- TickObject{t}:
					default:
					}
				}
			}
		}()

		return ipChans, signers, ticker, done, shard.ConsensusThreshold()
	}

	table := []struct {
		numHyperdrives int
		maxHeight      block.Height
	}{
		{1, 640},
		{2, 320},
		{4, 160},
		{8, 80},
		{16, 40},
		{32, 20},
		{64, 10},

		// Disabled so that the CI does not take too long.
		// {128, 5},
		// {256, 5},
		// {512, 5},
		// {1024, 5},
	}

	for _, entry := range table {
		entry := entry

		Context(fmt.Sprintf("when reaching consensus on a shard with %v replicas", entry.numHyperdrives), func() {
			It("should commit blocks", func() {
				// The estimated number of messages a Replica will receive throughout the test
				cap := 2 * (entry.numHyperdrives + 1) * int(entry.maxHeight)
				// Increase by an order of magnitude to account for timeouts and
				// multiple rounds
				cap = 10 * cap

				ipChans, signers, ticker, done, _ := initReplicas(entry.numHyperdrives)
				defer ticker.Stop()

				co.ParForAll(entry.numHyperdrives, func(i int) {
					defer GinkgoRecover()

					h := New(signers[i], NewMockDispatcher(i, ipChans, done, cap))
					Expect(runHyperdrive(i, h, ipChans[i], done, entry.maxHeight)).ShouldNot((HaveOccurred()))
				})
			})

			if entry.numHyperdrives > 2 && entry.numHyperdrives <= 32 {
				Context("when leader at index = 0 is inactive", func() {
					FIt("should commit blocks with new leader", func() {
						cap := 2 * (entry.numHyperdrives + 1) * int(entry.maxHeight)
						ipChans, signers, ticker, done, consensusThreshold := initReplicas(entry.numHyperdrives)
						defer ticker.Stop()

						co.ParForAll(entry.numHyperdrives, func(i int) {
							defer GinkgoRecover()

							h := New(signers[i], NewMockDispatcher(i, ipChans, done, cap))
							if i == 0 {
								h = testutils.NewFaultyLeader(signers[i], NewMockDispatcher(i, ipChans, done, cap), consensusThreshold)
							}

							Expect(runHyperdrive(i, h, ipChans[i], done, entry.maxHeight)).ShouldNot(HaveOccurred())
						})
					})
				})
			}
		})
	}
})

type mockDispatcher struct {
	index int

	dups     map[string]bool
	channels []chan Object
	reqCh    chan ActionObject

	done chan struct{}
}

func NewMockDispatcher(i int, channels []chan Object, done chan struct{}, cap int) *mockDispatcher {
	dispatcher := &mockDispatcher{
		index: i,

		dups:     map[string]bool{},
		channels: channels,
		reqCh:    make(chan ActionObject, cap),

		done: done,
	}

	go func() {
		for {
			select {
			case <-dispatcher.done:
				return
			case actionObject := <-dispatcher.reqCh:
				for i := range dispatcher.channels {
					select {
					case <-dispatcher.done:
						return
					case dispatcher.channels[i] <- actionObject:
					}
				}
			default:
			}
		}
	}()
	return dispatcher
}

func (mockDispatcher *mockDispatcher) Dispatch(shardHash sig.Hash, action state.Action) {

	// De-duplicate
	height := block.Height(0)
	round := block.Round(0)
	switch action := action.(type) {
	case state.Propose:
		height = action.Height
		round = action.Round
	case state.SignedPreVote:
		height = action.Height
		round = action.Round
	case state.SignedPreCommit:
		height = action.Polka.Height
		round = action.Polka.Round
	case state.Commit:
		height = action.Polka.Height
		round = action.Polka.Round
	default:
		panic(fmt.Errorf("unexpected action type %T", action))
	}

	key := fmt.Sprintf("Key(Shard=%v,Height=%v,Round=%v,Action=%T)", shardHash, height, round, action)
	if dup := mockDispatcher.dups[key]; dup {
		return
	}
	mockDispatcher.dups[key] = true

	select {
	case <-mockDispatcher.done:
		return
	case mockDispatcher.reqCh <- ActionObject{shardHash, action}:
	}
}

type Object interface {
	IsObject()
}

type ActionObject struct {
	shardHash sig.Hash
	action    state.Action
}

func (ActionObject) IsObject() {}

type ShardObject struct {
	shard shard.Shard
	pool  tx.Pool
}

func (ShardObject) IsObject() {}

type TickObject struct {
	Time time.Time
}

func (TickObject) IsObject() {}

func runHyperdrive(index int, h Hyperdrive, inputCh chan Object, done chan struct{}, maxHeight block.Height) error {
	var currentBlock *block.SignedBlock

	for {
		select {
		case <-done:
			return nil
		case input := <-inputCh:
			switch input := input.(type) {
			case TickObject:
				h.AcceptTick(input.Time)
			case ShardObject:
				h.BeginShard(input.shard, block.Genesis(), input.pool)
			case ActionObject:
				switch action := input.action.(type) {
				case state.Propose:
					h.AcceptPropose(input.shardHash, action.SignedPropose)
				case state.SignedPreVote:
					h.AcceptPreVote(input.shardHash, action.SignedPreVote)
				case state.SignedPreCommit:
					h.AcceptPreCommit(input.shardHash, action.SignedPreCommit)
				case state.Commit:
					Expect(len(action.Commit.Polka.Block.Txs)).To(BeNumerically("<=", block.MaxTransactions))
					if currentBlock == nil || action.Polka.Block.Height > currentBlock.Height {
						if currentBlock != nil {
							Expect(action.Polka.Block.Height).To(Equal(currentBlock.Height + 1))
							Expect(currentBlock.Header.Equal(action.Polka.Block.ParentHeader)).To(Equal(true))
						}
						if index == 0 {
							fmt.Printf("%v\n", *action.Polka.Block)
						}
						currentBlock = action.Polka.Block
						if currentBlock.Height == maxHeight {
							return nil
						}
					}
				default:
				}
			}
		}
	}
}

func rand32Byte() [32]byte {
	key := make([]byte, 32)

	rand.Read(key)
	b := [32]byte{}
	copy(b[:], key[:])
	return b
}

func populateTxPool(txPool tx.Pool) {
	for {
		tx := testutils.RandomTransaction()
		if err := txPool.Enqueue(tx); err != nil {
			time.Sleep(time.Duration(mrand.Intn(5)) * time.Millisecond)
		}
	}
}
