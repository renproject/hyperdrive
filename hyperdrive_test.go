package hyperdrive_test

import (
	"crypto/rand"
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
				// TODO: Done channel
				go func(i int, signer sig.SignerVerifier) {
					defer wg.Done()
					// participants := append(ipChans[:i], ipChans[i+1:]...)
					dispatcher := NewMockDispatcher(i, ipChans)

					runHyperdrive(i, dispatcher, signer, ipChans[i])
				}(i, signers[i])

			}
			time.Sleep(5 * time.Second)

			for i := range ipChans {

				blockchain := block.NewBlockchain()
				select {
				case ipChans[i] <- ShardObject{shard, blockchain, pool, i}:
				}
			}

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
	i          int
}

func (ShardObject) IsObject() {}

func runHyperdrive(index int, dispatcher replica.Dispatcher, signer sig.SignerVerifier, inputCh chan Object) {
	h := New(signer, dispatcher)
	shardsCh := make(chan ShardObject)

	proposeCh := make(chan ActionObject)
	preVoteCh := make(chan ActionObject)
	preCommitCh := make(chan ActionObject)

	acceptedShard := false
	co.ParBegin(
		func() {
			for {
				select {
				case shard := <-shardsCh:
					h.AcceptShard(shard.shard, shard.blockchain, shard.pool, shard.i)
					acceptedShard = true
				default:
				}
			}
		},
		func() {
			for {
				if acceptedShard {
					select {
					case propose := <-proposeCh:
						deepCopy := propose.action.(consensus.Propose)
						h.AcceptPropose(propose.shardHash, deepCopy.SignedBlock)
					default:
					}
				}
			}
		},
		func() {
			for {
				if acceptedShard {
					select {
					case preVote := <-preVoteCh:

						temp := preVote.action.(consensus.SignedPreVote)

						deepCopy := *temp.SignedPreVote.Block
						copy := temp

						copy.SignedPreVote.Block = &deepCopy

						h.AcceptPreVote(preVote.shardHash, copy.SignedPreVote)
					default:
					}
				}
			}
		},
		func() {
			for {
				if acceptedShard {
					select {
					case preCommit := <-preCommitCh:
						temp := preCommit.action.(consensus.SignedPreCommit)

						deepCopy := *temp.SignedPreCommit.Polka.Block
						copy := temp

						copy.SignedPreCommit.Polka.Block = &deepCopy

						h.AcceptPreCommit(preCommit.shardHash, copy.SignedPreCommit)
					default:
					}
				}
			}
		},
		func() {
			for {
				select {
				case input := <-inputCh:
					switch input := input.(type) {
					case ShardObject:
						select {
						case shardsCh <- input:
						}
					case ActionObject:
						switch input.action.(type) {
						case consensus.Propose:
							proposeCh <- input
						case consensus.SignedPreVote:
							preVoteCh <- input
						case consensus.SignedPreCommit:
							preCommitCh <- input
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
