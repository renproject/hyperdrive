package hyperdrive_test

import (
	"crypto/rand"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
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
	}{
		{8},
		{16},
		{32},
		{64},
		{128},
		{256},
		// CircleCI times out on the following configurations
		// {512},
		// {1024},
	}

	for _, entry := range table {
		entry := entry

		Context(fmt.Sprintf("when reaching consensus on a shard with %v replicas", entry.numHyperdrives), func() {
			It("should commit blocks", func() {
				done := make(chan struct{})
				ipChans := make([]chan Object, entry.numHyperdrives)
				signatories := make(sig.Signatories, entry.numHyperdrives)
				signers := make([]sig.SignerVerifier, entry.numHyperdrives)

				By("building signatories")
				for i := 0; i < entry.numHyperdrives; i++ {
					var err error
					ipChans[i] = make(chan Object, entry.numHyperdrives*entry.numHyperdrives)
					signers[i], err = ecdsa.NewFromRandom()
					signatories[i] = signers[i].Signatory()
					Expect(err).ShouldNot(HaveOccurred())
				}

				By("initialising shard")
				shardHash := testutils.RandomHash()
				for i := 0; i < entry.numHyperdrives; i++ {
					blockchain := block.NewBlockchain()
					shard := shard.Shard{
						Hash:        shardHash,
						Signatories: make(sig.Signatories, entry.numHyperdrives),
					}
					copy(shard.Signatories[:], signatories[:])
					ipChans[i] <- ShardObject{shard, blockchain, tx.FIFOPool()}
				}

				By("running hyperdrives")
				co.ParBegin(
					func() {
						defer close(done)
						timeout := math.Ceil(float64(entry.numHyperdrives)*0.1) + 1
						log.Println(timeout)
						time.Sleep(time.Duration(timeout) * time.Second)
					},
					func() {
						co.ParForAll(entry.numHyperdrives, func(i int) {
							if i == 0 {
								time.Sleep(time.Second)
							}
							log.Printf("running hyperdrive %v", i)
							runHyperdrive(i, NewMockDispatcher(i, ipChans, done), signers[i], ipChans[i], done)
						})
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

func (mockDispatcher *mockDispatcher) Dispatch(shardHash sig.Hash, action consensus.Action) {

	// De-duplicate
	height := block.Height(0)
	round := block.Round(0)
	switch action := action.(type) {
	case consensus.Propose:
		height = action.Height
		round = action.Round
	case consensus.SignedPreVote:
		height = action.Height
		round = action.Round
	case consensus.SignedPreCommit:
		height = action.Polka.Height
		round = action.Polka.Round
	case consensus.Commit:
		height = action.Polka.Height
		round = action.Polka.Round
	default:
		panic(fmt.Errorf("unexpected action type %T", action))
	}
	key := fmt.Sprintf("Key(Shard=%v,Height=%v,Round=%v,Action=%T)", shardHash, height, round, action)

	if dup := mockDispatcher.dups[key]; dup {
		return
	}
	if mockDispatcher.index == 0 {
		log.Printf("dispatching %v", key)
	}
	mockDispatcher.dups[key] = true

	go func() {
		for i := range mockDispatcher.channels {
			select {
			case <-mockDispatcher.done:
				return
			case mockDispatcher.channels[i] <- ActionObject{shardHash, action}:
			}
		}
	}()
}

type Object interface {
	IsObject()
}

type ActionObject struct {
	shardHash sig.Hash
	action    consensus.Action
}

func (ActionObject) IsObject() {}

type ShardObject struct {
	shard      shard.Shard
	blockchain block.Blockchain
	pool       tx.Pool
}

func (ShardObject) IsObject() {}

func runHyperdrive(index int, dispatcher replica.Dispatcher, signer sig.SignerVerifier, inputCh chan Object, done chan struct{}) {
	h := New(signer, dispatcher)

	currentHeight := block.Height(-1)

	for {
		select {
		case <-done:
			return
		case input := <-inputCh:
			switch input := input.(type) {
			case ShardObject:
				select {
				case <-done:
					return
				default:
					h.AcceptShard(input.shard, input.blockchain, input.pool)
				}
			case ActionObject:
				switch input.action.(type) {
				case consensus.Propose:
					deepCopy := input.action.(consensus.Propose)
					select {
					case <-done:
						return
					default:
						h.AcceptPropose(input.shardHash, deepCopy.SignedBlock)
					}
				case consensus.SignedPreVote:
					temp := input.action.(consensus.SignedPreVote)
					if temp.Block.Height > currentHeight {
						deepCopy := *temp.SignedPreVote.Block
						copy := temp

						copy.SignedPreVote.Block = &deepCopy
						select {
						case <-done:
							return
						default:
							h.AcceptPreVote(input.shardHash, copy.SignedPreVote)
						}
					}
				case consensus.SignedPreCommit:
					temp := input.action.(consensus.SignedPreCommit)
					if temp.SignedPreCommit.Polka.Block.Height > currentHeight {
						deepCopy := *temp.SignedPreCommit.Polka.Block
						copy := temp

						copy.SignedPreCommit.Polka.Block = &deepCopy
						select {
						case <-done:
							return
						default:
							h.AcceptPreCommit(input.shardHash, copy.SignedPreCommit)
						}
					}
				case consensus.Commit:
					if input.action.(consensus.Commit).Polka.Block.Height > currentHeight {
						if index == 0 {
							log.Printf("got commit %x\ncurrent height %d; new height %d\n", input.action.(consensus.Commit).Polka.Block.Header, currentHeight, input.action.(consensus.Commit).Polka.Block.Height)
						}
						currentHeight = input.action.(consensus.Commit).Polka.Block.Height
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
