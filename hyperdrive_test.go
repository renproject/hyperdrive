package hyperdrive_test

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/testutils"
	"github.com/renproject/hyperdrive/tx"
	"github.com/republicprotocol/co-go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive"
)

var _ = Describe("Hyperdrive", func() {

	Context("when ", func() {
		It("should ", func() {
			ipChans := make([]chan Object, 5)
			signatories := make(sig.Signatories, 5)
			signers := make([]sig.SignerVerifier, 5)
			// blockchains := make([]block.Blockchain, 5)
			pool := tx.FIFOPool()

			done := make(chan struct{})

			var err error
			var wg sync.WaitGroup
			for i := 0; i < 5; i++ {
				ipChans[i] = make(chan Object, 100)

				signers[i], err = ecdsa.NewFromRandom()
				Expect(err).ShouldNot(HaveOccurred())
				signatories[i] = signers[i].Signatory()
			}
			shard := shard.Shard{
				Hash:        testutils.RandomHash(),
				BlockHeader: sig.Hash{},
				BlockHeight: 1,
				Signatories: signatories,
			}
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(i int, signer sig.SignerVerifier) {
					defer wg.Done()
					// participants := append(ipChans[:i], ipChans[i+1:]...)
					dispatcher := NewMockDispatcher(i, ipChans)

					runHyperdrive(i, dispatcher, signer, ipChans[i], done)
				}(i, signers[i])

			}
			time.Sleep(5 * time.Second)

			for i := range ipChans {

				blockchain := block.NewBlockchain()
				select {
				case ipChans[i] <- ShardObject{shard, blockchain, pool}:
				}
			}

			go func() {
				time.Sleep(10 * time.Second)
				close(done)
			}()

			wg.Wait()
		})
	})
})

type mockDispatcher struct {
	index      int
	channelsMu *sync.Mutex
	channels   []chan Object
}

func NewMockDispatcher(i int, channels []chan Object) *mockDispatcher {
	return &mockDispatcher{
		index:      i,
		channelsMu: new(sync.Mutex),
		channels:   channels,
	}
}

func (mockDispatcher *mockDispatcher) Dispatch(shardHash sig.Hash, action consensus.Action) {
	mockDispatcher.channelsMu.Lock()
	defer mockDispatcher.channelsMu.Unlock()

	for i := range mockDispatcher.channels {
		select {
		case mockDispatcher.channels[i] <- ActionObject{shardHash, action}:
		}
	}
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

	co.ParBegin(
		func() {
			for {
				select {
				case <-done:
					return
				case input := <-inputCh:
					switch input := input.(type) {
					case ShardObject:
						h.AcceptShard(input.shard, input.blockchain, input.pool)
					case ActionObject:
						switch input.action.(type) {
						case consensus.Propose:
							deepCopy := input.action.(consensus.Propose)
							h.AcceptPropose(input.shardHash, deepCopy.SignedBlock)
						case consensus.SignedPreVote:
							temp := input.action.(consensus.SignedPreVote)
							if temp.Block.Height > currentHeight {
								deepCopy := *temp.SignedPreVote.Block
								copy := temp

								copy.SignedPreVote.Block = &deepCopy

								h.AcceptPreVote(input.shardHash, copy.SignedPreVote)
							}
						case consensus.SignedPreCommit:
							temp := input.action.(consensus.SignedPreCommit)
							if temp.SignedPreCommit.Polka.Block.Height > currentHeight {
								deepCopy := *temp.SignedPreCommit.Polka.Block
								copy := temp

								copy.SignedPreCommit.Polka.Block = &deepCopy

								h.AcceptPreCommit(input.shardHash, copy.SignedPreCommit)
							}
						case consensus.Commit:
							if input.action.(consensus.Commit).Polka.Block.Height > currentHeight {
								if index == 0 {
									fmt.Printf("got commit %x\ncurrent height %d; new height %d\n", input.action.(consensus.Commit).Polka.Block.Header, currentHeight, input.action.(consensus.Commit).Polka.Block.Height)
								}
								currentHeight = input.action.(consensus.Commit).Polka.Block.Height
							}
						default:
						}
					}
				}
			}
		},
	)
	return
}

func rand32Byte() [32]byte {
	key := make([]byte, 32)

	rand.Read(key)
	b := [32]byte{}
	copy(b[:], key[:])
	return b
}
