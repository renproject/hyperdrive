package replica_test

import (
	"fmt"
	"reflect"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/state"
	"github.com/renproject/hyperdrive/testutils"
	"github.com/renproject/hyperdrive/tx"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/replica"
)

var _ = Describe("Replica", func() {

	Context("when Init is called", func() {
		It("should generate a new block", func() {
			transitionBuffer := state.NewTransitionBuffer(128)
			pool := tx.FIFOPool(100)
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			shard := shard.Shard{
				Hash:        sig.Hash{},
				BlockHeader: sig.Hash{},
				BlockHeight: 0,
				Signatories: sig.Signatories{signer.Signatory()},
			}
			stateMachine := state.NewMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), 1)

			blockchain := block.NewBlockchain()

			replica := New(newMockDispatcher(), signer, pool, state.WaitForPropose(0, 0), stateMachine, transitionBuffer, shard, &blockchain)
			Expect(func() { replica.Init() }).ToNot(Panic())
		})
	})

	Context("when new Transitions are sent", func() {

		signer, err := ecdsa.NewFromRandom()
		if err != nil {
			panic(fmt.Sprintf("error generating random SignerVerifier: %v", err))
		}
		participant1, err := ecdsa.NewFromRandom()
		if err != nil {
			panic(fmt.Sprintf("error generating random SignerVerifier: %v", err))
		}
		participant2, err := ecdsa.NewFromRandom()
		if err != nil {
			panic(fmt.Sprintf("error generating random SignerVerifier: %v", err))
		}
		testCases := generateTestCases(signer, participant1, participant2)
		for _, t := range testCases {
			t := t

			Context(fmt.Sprintf("when replica starts with intial state - %s", reflect.TypeOf(t.startingState).Name()), func() {
				It(fmt.Sprintf("should arrive at %s", reflect.TypeOf(t.finalState).Name()), func() {

					transitionBuffer := state.NewTransitionBuffer(128)
					pool := tx.FIFOPool(100)

					for i := 0; i < 100; i++ {
						tx := testutils.RandomTransaction()
						pool.Enqueue(tx)
					}

					shard := shard.Shard{
						Hash:        sig.Hash{},
						BlockHeader: sig.Hash{},
						BlockHeight: 0,
						Signatories: sig.Signatories{signer.Signatory(), participant1.Signatory(), participant2.Signatory()},
					}
					stateMachine := state.NewMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), t.consensusThreshold)
					blockchain := block.NewBlockchain()

					replica := New(NewMockDispatcher(), signer, pool, t.startingState, stateMachine, transitionBuffer, shard, &blockchain)
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
	consensusThreshold int

	startingState state.State
	finalState    state.State

	transitions []state.Transition
}

func generateTestCases(signer, p1, p2 sig.SignerVerifier) []TestCase {
	signedBlock := testutils.SignBlock(block.Block{
		Height: 1,
		Header: testutils.RandomHash(),
	}, signer)
	signedFutureBlock := testutils.SignBlock(block.Block{
		Height:       2,
		Header:       testutils.RandomHash(),
		ParentHeader: signedBlock.Header,
	}, signer)

	maliciousSigner, err := ecdsa.NewFromRandom()
	if err != nil {
		panic(fmt.Sprintf("error generating random SignerVerifier: %v", err))
	}

	return []TestCase{
		{
			consensusThreshold: 2,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Height: 1,
								Block: testutils.SignBlock(block.Block{
									Round:        2,
									Header:       testutils.RandomHash(),
									ParentHeader: testutils.RandomHash(),
								}, signer),
								Signatories: testutils.RandomSignatories(3),
								Signatures:  testutils.RandomSignatures(3),
							},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 2,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Height: 1,
								Block: testutils.SignBlock(block.Block{
									Height:       2,
									Header:       testutils.RandomHash(),
									ParentHeader: testutils.RandomHash(),
								}, signer),
								Signatories: testutils.RandomSignatories(3),
								Signatures:  testutils.RandomSignatures(3),
							},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 2,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Height: 1,
								Block: testutils.SignBlock(block.Block{
									Height:       1,
									Header:       testutils.RandomHash(),
									ParentHeader: testutils.RandomHash(),
								}, signer),
								Signatories: testutils.RandomSignatories(3),
								Signatures:  testutils.RandomSignatures(3),
							},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 2,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Height:      1,
								Block:       signedBlock,
								Signatories: testutils.RandomSignatories(3),
								Signatures:  testutils.RandomSignatures(3),
							},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 2,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Height:      1,
								Block:       signedBlock,
								Signatories: testutils.RandomSignatories(1),
								Signatures:  testutils.RandomSignatures(1),
							},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Height:      1,
								Block:       signedBlock,
								Signatories: testutils.RandomSignatories(3),
								Signatures:  testutils.RandomSignatures(2),
							},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, -1),
			finalState:    state.WaitForPropose(0, -1),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Round:  1,
								Height: -1,
								Block:  signedBlock,
							},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Round:  -1,
								Height: 1,
								Block:  signedBlock,
							},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPropose(0, 0),

			transitions: []state.Transition{
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{},
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreVoted{SignedPreVote: testutils.GenerateSignedPreVote(*signedBlock, maliciousSigner)},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 2),
			finalState:    state.WaitForPropose(0, 2),

			transitions: []state.Transition{
				state.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Block:  signedBlock,
							Height: 2,
							Round:  0,
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Block:  signedBlock,
							Height: 1,
							Round:  1,
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Block: testutils.SignBlock(block.Block{
								Height:       1,
								Header:       testutils.RandomHash(),
								ParentHeader: testutils.RandomHash(),
							}, signer),
							Height: 1,
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: *testutils.SignBlock(block.Block{
						Height:       1,
						Header:       testutils.RandomHash(),
						ParentHeader: testutils.RandomHash(),
					}, signer),
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: *testutils.SignBlock(block.Block{Height: 1}, maliciousSigner),
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPropose(0, 0),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{
							Height: -1,
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPropose(0, 0),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{
							Round: -1,
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPropose(0, 1),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{
							Height: 1,
							Time:   time.Now().Add(10 * time.Minute),
						},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPropose(0, 0),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{
							Height: 0,
							Header: testutils.RandomHash(),
						},
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPropose(0, 0),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(1, 1),
			finalState:    state.WaitForPropose(1, 1),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: *signedBlock,
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPropose(0, 0),

			transitions: []state.Transition{testutils.InvalidTransition{}},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPropose(0, 0),

			transitions: []state.Transition{
				state.TimedOut{Time: time.Now().Add(10 * time.Minute)},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPolka(0, 0),

			transitions: []state.Transition{
				state.TimedOut{Time: time.Now()},
			},
		},

		{
			consensusThreshold: 1,

			startingState: state.WaitForPropose(1, 0),
			finalState:    state.WaitForPropose(1, 0),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{},
					},
				},
			},
		},

		{
			consensusThreshold: 2,

			startingState: state.WaitForPropose(0, 0),
			finalState:    state.WaitForPropose(0, 0),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{
							Height: -1,
						},
					},
				},
				state.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Height: -1,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				state.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Height: -1,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &block.SignedBlock{
									Block: block.Block{
										Height: -1,
									},
								},
								Height: -1,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				state.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &block.SignedBlock{
									Block: block.Block{
										Height: -1,
									},
								},
								Height: -1,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},

		{
			consensusThreshold: 3,

			startingState: state.WaitForPropose(0, 1),
			finalState:    state.WaitForPolka(0, 3),

			transitions: []state.Transition{
				state.Proposed{
					SignedBlock: *signedFutureBlock,
				},
				state.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedFutureBlock, signer),
				},
				state.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedFutureBlock, p1),
				},
				state.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedFutureBlock, p2),
				},
				state.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedFutureBlock, signer, []sig.SignerVerifier{signer, p1, p2}),
				},
				state.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedFutureBlock, p1, []sig.SignerVerifier{signer, p1, p2}),
				},
				state.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedFutureBlock, p2, []sig.SignerVerifier{signer, p1, p2}),
				},
				state.Proposed{
					SignedBlock: *signedBlock,
				},
				state.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedBlock, signer),
				},
				state.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedBlock, p1),
				},
				state.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedBlock, p2),
				},
				state.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedBlock, signer, []sig.SignerVerifier{signer, p1, p2}),
				},
				state.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedBlock, p1, []sig.SignerVerifier{signer, p1, p2}),
				},
				state.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedBlock, p2, []sig.SignerVerifier{signer, p1, p2}),
				},
			},
		},
	}
}

type mockDispatcher struct{}

func newMockDispatcher() *mockDispatcher {
	return &mockDispatcher{}
}

func (mockDispatcher *mockDispatcher) Dispatch(shardHash sig.Hash, action state.Action) {}

type MockDispatcher struct {
}

func NewMockDispatcher() Dispatcher {
	return &MockDispatcher{}
}

func (dispatcher *MockDispatcher) Dispatch(shardHash sig.Hash, action state.Action) {}
