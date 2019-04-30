package hyperdrive_test

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/replica"
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
				done := make(chan struct{})
				ipChans := make([]chan Object, entry.numHyperdrives)
				signatories := make(sig.Signatories, entry.numHyperdrives)
				signers := make([]sig.SignerVerifier, entry.numHyperdrives)

				for i := 0; i < entry.numHyperdrives; i++ {
					var err error
					ipChans[i] = make(chan Object, entry.numHyperdrives*entry.numHyperdrives)
					signers[i], err = ecdsa.NewFromRandom()
					signatories[i] = signers[i].Signatory()
					Expect(err).ShouldNot(HaveOccurred())
				}

				txPool := tx.FIFOPool(100)

				go populateTxPool(txPool)

				shardHash := testutils.RandomHash()
				for i := 0; i < entry.numHyperdrives; i++ {
					shard := shard.Shard{
						Hash:        shardHash,
						Signatories: make(sig.Signatories, entry.numHyperdrives),
					}
					copy(shard.Signatories[:], signatories[:])
					ipChans[i] <- ShardObject{shard, txPool}
				}

				co.ParForAll(entry.numHyperdrives, func(i int) {
					defer GinkgoRecover()

					if i == 0 {
						time.Sleep(time.Second)
					}
					runHyperdrive(i, NewMockDispatcher(i, ipChans, done), signers[i], ipChans[i], done, entry.maxHeight)
				})
			})
		})
	}
})

type mockDispatcher struct {
	index int

	dups     map[string]bool
	channels []chan Object
	done     chan struct{}
}

func NewMockDispatcher(i int, channels []chan Object, done chan struct{}) *mockDispatcher {
	return &mockDispatcher{
		index: i,

		dups:     map[string]bool{},
		channels: channels,

		done: done,
	}
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

	for i := range mockDispatcher.channels {
		i := i
		go func() {
			select {
			case <-mockDispatcher.done:
				return
			case mockDispatcher.channels[i] <- ActionObject{shardHash, action}:
			}
		}()
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

func runHyperdrive(index int, dispatcher replica.Dispatcher, signer sig.SignerVerifier, inputCh chan Object, done chan struct{}, maxHeight block.Height) {
	h := New(signer, dispatcher)

	var currentBlock *block.SignedBlock

	for {
		select {
		case <-done:
			return
		case input := <-inputCh:
			switch input := input.(type) {
			case ShardObject:
				h.AcceptShard(input.shard, block.Genesis(), input.pool)
			case ActionObject:
				switch action := input.action.(type) {
				case state.Propose:
					h.AcceptPropose(input.shardHash, action.SignedBlock)
				case state.SignedPreVote:
					h.AcceptPreVote(input.shardHash, action.SignedPreVote)
				case state.SignedPreCommit:
					h.AcceptPreCommit(input.shardHash, action.SignedPreCommit)
				case state.Commit:
					Expect(len(action.Commit.Polka.Block.Txs)).To(Equal(block.MaxTransactions))
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
							return
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
