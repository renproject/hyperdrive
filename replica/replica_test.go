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

			replica := New(hyperdrive.NewDispatcher(shard), signer, pool, consensus.WaitForPropose(0, 0), stateMachine, transitionBuffer, block.NewBlockchain(), shard)
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

			replica := New(hyperdrive.NewDispatcher(shard), signer, pool, consensus.WaitForPropose(0, 0), stateMachine, transitionBuffer, block.NewBlockchain(), shard)
			replica.Transact(tx.Transaction{})
			transaction, ok := pool.Dequeue()
			Expect(ok).To(BeTrue())
			Expect(transaction).Should(Equal(tx.Transaction{}))
		})
	})

	Context("when new Transitions are sent", func() {

		signer, _ := ecdsa.NewFromRandom()
		p1, _ := ecdsa.NewFromRandom()
		p2, _ := ecdsa.NewFromRandom()
		testCases := generateTestCases(signer, p1, p2)
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
						Signatories: sig.Signatories{signer.Signatory(), p1.Signatory(), p2.Signatory()},
					}
					stateMachine := consensus.NewStateMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), t.consensusThreshold)

					replica := New(hyperdrive.NewDispatcher(shard), signer, pool, t.startingState, stateMachine, transitionBuffer, block.NewBlockchain(), shard)
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
	signedFutureBlock := signBlock(block.Block{
		Height: 2,
		Header: testutils.RandomHash(),
	}, signer)
	signedBlock := signBlock(block.Block{
		Height: 1,
		Header: testutils.RandomHash(),
	}, signer)

	return []TestCase{

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(1, 1),
			finalState:    consensus.WaitForPropose(1, 1),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: signedBlock,
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
				consensus.TimedOut{time.Now().Add(10 * time.Minute)},
			},
		},

		{
			consensusThreshold: 1,

			startingState: consensus.WaitForPropose(0, 0),
			finalState:    consensus.WaitForPolka(0, 0),

			transitions: []consensus.Transition{
				consensus.TimedOut{time.Now()},
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
			consensusThreshold: 2,

			startingState: consensus.WaitForPropose(0, 1),
			finalState:    consensus.WaitForPropose(0, 3),

			transitions: []consensus.Transition{
				consensus.Proposed{
					SignedBlock: signedFutureBlock,
				},
				consensus.PreVoted{
					SignedPreVote: generateSignedPreVote(signedFutureBlock, p1),
				},
				consensus.PreVoted{
					SignedPreVote: generateSignedPreVote(signedFutureBlock, p2),
				},
				consensus.PreCommitted{
					SignedPreCommit: generateSignedPreCommit(signedFutureBlock, p1, p1, p2),
				},
				consensus.PreCommitted{
					SignedPreCommit: generateSignedPreCommit(signedFutureBlock, p2, p1, p2),
				},
				consensus.Proposed{
					SignedBlock: signedBlock,
				},
				consensus.PreVoted{
					SignedPreVote: generateSignedPreVote(signedBlock, p1),
				},
				consensus.PreVoted{
					SignedPreVote: generateSignedPreVote(signedBlock, p2),
				},
				consensus.PreCommitted{
					SignedPreCommit: generateSignedPreCommit(signedBlock, p1, p1, p2),
				},
				consensus.PreCommitted{
					SignedPreCommit: generateSignedPreCommit(signedBlock, p2, p1, p2),
				},
			},
		},
	}
}

func signBlock(blk block.Block, signer sig.SignerVerifier) block.SignedBlock {
	signedBlock, _ := blk.Sign(signer)
	return signedBlock
}

func generateSignedPreVote(signedBlock block.SignedBlock, signer sig.SignerVerifier) block.SignedPreVote {
	preVote := block.PreVote{
		Block:  &signedBlock,
		Height: signedBlock.Height,
	}
	signedPreVote, _ := preVote.Sign(signer)
	return signedPreVote
}

func generateSignedPreCommit(signedBlock block.SignedBlock, signer, p1, p2 sig.SignerVerifier) block.SignedPreCommit {
	signedPreVote1 := generateSignedPreVote(signedBlock, p1)
	signedPreVote2 := generateSignedPreVote(signedBlock, p2)

	signatures := []sig.Signature{signedPreVote1.Signature, signedPreVote2.Signature}
	signatories := []sig.Signatory{signedPreVote1.Signatory, signedPreVote2.Signatory}

	preCommit := block.PreCommit{
		Polka: block.Polka{
			Block:       &signedBlock,
			Height:      signedBlock.Height,
			Signatures:  signatures,
			Signatories: signatories,
		},
	}

	signedPreCommit, _ := preCommit.Sign(signer)
	return signedPreCommit
}
