package replica_test

import (
	"fmt"
	"reflect"

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

			replica := New(testutils.NewMockDispatcher(), signer, pool, consensus.WaitForPropose(0, 0), stateMachine, transitionBuffer, &blockchain, shard)
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

			replica := New(testutils.NewMockDispatcher(), signer, pool, consensus.WaitForPropose(0, 0), stateMachine, transitionBuffer, &blockchain, shard)
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

			FContext(fmt.Sprintf("when replica starts with intial state - %s", reflect.TypeOf(t.startingState).Name()), func() {
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
	// futureBlockHeader := testutils.RandomHash()
	// signedFutureBlock := signBlock(block.Block{
	// 	Height: 2,
	// 	Header: futureBlockHeader,
	// }, signer)
	// blockHeader := testutils.RandomHash()
	// signedBlock := signBlock(block.Block{
	// 	Height: 1,
	// 	Header: blockHeader,
	// }, signer)

	return []TestCase{
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

		/*{
			consensusThreshold: 2,

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 3),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: signedFutureBlock,
				},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 2,
									Header: futureBlockHeader,
								},
							},
							Height: 2,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 2,
									Header: futureBlockHeader,
								},
							},
							Height: 2,
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
										Height: 2,
										Header: futureBlockHeader,
									},
								},
								Height: 2,
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
								Block:  &signedFutureBlock,
								Height: 2,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				consensus.Proposed{
					SignedBlock: signedBlock,
				},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Block:  &signedBlock,
							Height: 1,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				consensus.PreVoted{
					SignedPreVote: block.SignedPreVote{
						PreVote: block.PreVote{
							Block:  &signedBlock,
							Height: 1,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				consensus.PreCommitted{
					SignedPreCommit: block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block:  &signedBlock,
								Height: 1,
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
								Block:  &signedBlock,
								Height: 1,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},*/
	}
}

func signBlock(blk block.Block, signer sig.SignerVerifier) block.SignedBlock {
	signedBlock, _ := blk.Sign(signer)
	return signedBlock
}

// func signPreVote(blk block.PreVote, signer sig.SignerVerifier) block.SignedBlock {
// 	signedBlock, _ := blk.Sign(signer)
// 	return signedBlock
// }
