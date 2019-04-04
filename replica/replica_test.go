package replica_test

import (
	"fmt"
	"reflect"
	"time"

	"github.com/renproject/hyperdrive"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/testutils"
	"github.com/renproject/hyperdrive/tx"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/replica"
)

var _ = Describe("Replica", func() {

	Context("when Init is called", func() {
		It("should generate a new block", func() {
			transitionBuffer := consensus.NewTransitionBuffer(128)
			pool := tx.FIFOPool()
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			shard := shard.Shard{
				Hash:        sig.Hash{},
				BlockHeader: sig.Hash{},
				BlockHeight: 0,
				Signatories: sig.Signatories{signer.Signatory()},
			}
			stateMachine := consensus.NewStateMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), 1)
			blockchain := block.NewBlockchain()

			replica := New(hyperdrive.NewDispatcher(shard), signer, pool, consensus.WaitForPropose(0, 0), stateMachine, transitionBuffer, &blockchain, shard)
			Expect(func() { replica.Init() }).ToNot(Panic())
		})
	})

	Context("when a new Transaction is sent using Transact", func() {
		It("should update the TxPool", func() {
			transitionBuffer := consensus.NewTransitionBuffer(128)
			pool := tx.FIFOPool()
			signer, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())
			shard := shard.Shard{
				Hash:        sig.Hash{},
				BlockHeader: sig.Hash{},
				BlockHeight: 0,
				Signatories: sig.Signatories{signer.Signatory()},
			}
			stateMachine := consensus.NewStateMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), 1)
			blockchain := block.NewBlockchain()

			replica := New(hyperdrive.NewDispatcher(shard), signer, pool, consensus.WaitForPropose(0, 0), stateMachine, transitionBuffer, &blockchain, shard)
			replica.Transact(tx.Transaction{})
			transaction, ok := pool.Dequeue()
			Expect(ok).To(BeTrue())
			Expect(transaction).Should(Equal(tx.Transaction{}))
		})
	})

	Context("when new Transitions are sent", func() {

		signer, _ := ecdsa.NewFromRandom()
		participant1, _ := ecdsa.NewFromRandom()
		participant2, _ := ecdsa.NewFromRandom()
		testCases := generateTestCases(signer, participant1, participant2)
		for _, t := range testCases {
			t := t

			Context(fmt.Sprintf("when replica starts with intial state - %s", reflect.TypeOf(t.startingState).Name()), func() {
				It(fmt.Sprintf("should arrive at %s", reflect.TypeOf(t.finalState).Name()), func() {

					transitionBuffer := consensus.NewTransitionBuffer(128)
					pool := tx.FIFOPool()
					shard := shard.Shard{
						Hash:        sig.Hash{},
						BlockHeader: sig.Hash{},
						BlockHeight: 0,
						Signatories: sig.Signatories{signer.Signatory(), participant1.Signatory(), participant2.Signatory()},
					}
					stateMachine := consensus.NewStateMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), t.consensusThreshold)
					blockchain := block.NewBlockchain()

					replica := New(testutils.NewMockDispatcher(), signer, pool, t.startingState, stateMachine, transitionBuffer, &blockchain, shard)
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

	startingState consensus.State
	finalState    consensus.State

	transitions []consensus.Transition
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

	maliciousSigner, _ := ecdsa.NewFromRandom()

	return []TestCase{
		{
			consensusThreshold: 2,

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, -1),
			finalState:    consensus.WaitForPropose(0, -1),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 0),

			transitions: []consensus.Transition{
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreVoted{SignedPreVote: testutils.GenerateSignedPreVote(*signedBlock, maliciousSigner)},
			},
		},

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(0, 2),
			finalState:    consensus.WaitForPropose(0, 2),

			transitions: []consensus.Transition{
				consensus.PreVoted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreVoted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.PreVoted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.Proposed{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: *testutils.SignBlock(block.Block{Height: 1}, maliciousSigner),
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 0),

			transitions: []consensus.Transition{
				consensus.Proposed{
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

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 0),

			transitions: []consensus.Transition{
				consensus.Proposed{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 1),

			transitions: []consensus.Transition{
				consensus.Proposed{
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

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 0),

			transitions: []consensus.Transition{
				consensus.Proposed{
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

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 0),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{},
					},
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(1, 1),
			finalState:    consensus.WaitForPropose(1, 1),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: *signedBlock,
				},
			},
		},

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 0),

			transitions: []consensus.Transition{testutils.InvalidTransition{}},
		},

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 0),

			transitions: []consensus.Transition{
				consensus.TimedOut{Time: time.Now().Add(10 * time.Minute)},
			},
		},

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPolka(0, 0),

			transitions: []consensus.Transition{
				consensus.TimedOut{Time: time.Now()},
			},
		},

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(1, 0),
			finalState:    consensus.WaitForPropose(1, 0),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{},
					},
				},
			},
		},

		{
			consensusThreshold: 2,

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPropose(0, 0),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{
							Height: -1,
						},
					},
				},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Height: -1,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Height: -1,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				consensus.PreCommitted{
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
				consensus.PreCommitted{
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

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 3),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: *signedFutureBlock,
				},
				consensus.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedFutureBlock, signer),
				},
				consensus.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedFutureBlock, p1),
				},
				consensus.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedFutureBlock, p2),
				},
				consensus.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedFutureBlock, signer, []sig.SignerVerifier{signer, p1, p2}),
				},
				consensus.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedFutureBlock, p1, []sig.SignerVerifier{signer, p1, p2}),
				},
				consensus.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedFutureBlock, p2, []sig.SignerVerifier{signer, p1, p2}),
				},
				consensus.Proposed{
					SignedBlock: *signedBlock,
				},
				consensus.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedBlock, signer),
				},
				consensus.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedBlock, p1),
				},
				consensus.PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(*signedBlock, p2),
				},
				consensus.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedBlock, signer, []sig.SignerVerifier{signer, p1, p2}),
				},
				consensus.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedBlock, p1, []sig.SignerVerifier{signer, p1, p2}),
				},
				consensus.PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(*signedBlock, p2, []sig.SignerVerifier{signer, p1, p2}),
				},
			},
		},
	}
}
