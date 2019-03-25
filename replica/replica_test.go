package replica_test

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/tx"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/replica"
)

var _ = Describe("Replica", func() {

	BeforeSuite(func() {
		transitionBuffer = newMockTransitionBuffer()
		pool = NewMockLifoPool()
		dispatcher = newMockDispatcher()
	})

	Context("when a new Transaction is sent using Transact", func() {

		It("should update the TxPool", func() {
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			shard := shard.Shard{
				Hash:        sig.Hash{},
				BlockHeader: sig.Hash{},
				BlockHeight: 0,
				Signatories: sig.Signatories{signer.Signatory()},
			}
			stateMachine := consensus.NewStateMachine(block.PolkaBuilder{}, block.CommitBuilder{}, 1)

			replica := New(dispatcher, signer, pool, consensus.WaitForPropose(0, 0), stateMachine, transitionBuffer, block.NewBlockchain(), shard)
			replica.Transact(tx.Transaction{})
			Expect(pool.Length()).Should(Equal(1))
		})

	})

	Context("when new Transitions are sent", func() {

		testCases := generateTestCases()
		for _, t := range testCases {
			t := t

			Context(fmt.Sprintf("when the replica gets transition - %s", reflect.TypeOf(t.startingState).Name()), func() {
				It("should ", func() {
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					shard := shard.Shard{
						Hash:        sig.Hash{},
						BlockHeader: sig.Hash{},
						BlockHeight: 0,
						Signatories: sig.Signatories{signer.Signatory()},
					}
					stateMachine := consensus.NewStateMachine(block.PolkaBuilder{}, block.CommitBuilder{}, 1)

					replica := New(dispatcher, signer, pool, t.startingState, stateMachine, transitionBuffer, block.NewBlockchain(), shard)
					for _, transition := range t.transitions {
						replica.Transition(transition)
					}
					Expect(replica.State()).To(Equal(t.finalState))
				})
			})
		}

	})
})

type TestCase struct {
	startingState consensus.State
	finalState    consensus.State

	transitions []consensus.Transition
}

var dispatcher *mockDispatcher
var transitionBuffer *mockTransitionBuffer
var pool *mockLifoPool

func generateTestCases() []TestCase {

	return []TestCase{
		{
			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.Proposed{
					Block: block.Block{
						Height: -1,
					}},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Height: -1,
						},
					},
				},
				consensus.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block:  &block.Block{},
								Height: -1,
							},
						},
					},
				},
				consensus.Proposed{
					Block: block.Block{
						Height: 1,
					},
				},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Height: 1,
						},
					},
				},
				consensus.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block:  &block.Block{},
								Height: 1,
							},
						},
					},
				},
				consensus.Proposed{
					Block: block.Block{
						Height: 0,
					},
				},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Height: 0,
						},
					},
				},
				consensus.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block:  &block.Block{},
								Height: 0,
							},
						},
					},
				},
			},
		},
	}
}

type mockDispatcher struct {
	actionsMu *sync.Mutex
	actions   []consensus.Action
}

func newMockDispatcher() *mockDispatcher {
	return &mockDispatcher{
		new(sync.Mutex),
		[]consensus.Action{},
	}
}

func (mockDispatcher *mockDispatcher) Dispatch(action consensus.Action) {
	mockDispatcher.actionsMu.Lock()
	defer mockDispatcher.actionsMu.Unlock()

	mockDispatcher.actions = append(mockDispatcher.actions, action)
}

func (mockDispatcher *mockDispatcher) BufferLength() int {
	mockDispatcher.actionsMu.Lock()
	defer mockDispatcher.actionsMu.Unlock()

	return len(mockDispatcher.actions)
}

type mockTransitionBuffer struct {
	transitionsMu *sync.Mutex
	transitions   []consensus.Transition
}

func newMockTransitionBuffer() *mockTransitionBuffer {
	return &mockTransitionBuffer{
		new(sync.Mutex),
		[]consensus.Transition{},
	}
}

func (mockTransitionBuffer *mockTransitionBuffer) Enqueue(transition consensus.Transition) {
	mockTransitionBuffer.transitionsMu.Lock()
	defer mockTransitionBuffer.transitionsMu.Unlock()

	mockTransitionBuffer.transitions = append(mockTransitionBuffer.transitions, transition)
}

func (mockTransitionBuffer *mockTransitionBuffer) Dequeue(h block.Height) (consensus.Transition, bool) {
	mockTransitionBuffer.transitionsMu.Lock()
	defer mockTransitionBuffer.transitionsMu.Unlock()

	if len(mockTransitionBuffer.transitions) > 0 {
		tx := mockTransitionBuffer.transitions[len(mockTransitionBuffer.transitions)-1]
		mockTransitionBuffer.transitions = mockTransitionBuffer.transitions[:len(mockTransitionBuffer.transitions)-1]
		return tx, true
	}
	return nil, false
}

func (mockTransitionBuffer *mockTransitionBuffer) Drop(height block.Height) {

}

func (mockTransitionBuffer *mockTransitionBuffer) BufferLength() int {
	mockTransitionBuffer.transitionsMu.Lock()
	defer mockTransitionBuffer.transitionsMu.Unlock()

	return len(mockTransitionBuffer.transitions)
}

type mockLifoPool struct {
	txs []tx.Transaction
}

func NewMockLifoPool() *mockLifoPool {
	return &mockLifoPool{[]tx.Transaction{}}
}

func (pool *mockLifoPool) Enqueue(tx tx.Transaction) {
	pool.txs = append(pool.txs, tx)
}

func (pool *mockLifoPool) Dequeue() (tx.Transaction, bool) {
	if len(pool.txs) > 0 {
		tx := pool.txs[len(pool.txs)-1]
		pool.txs = pool.txs[:len(pool.txs)-1]
		return tx, true
	}
	return tx.Transaction{}, false
}

func (pool *mockLifoPool) Length() int {
	return len(pool.txs)
}
