package state_test

import (
	"fmt"
	"reflect"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/testutils"
	"github.com/renproject/hyperdrive/tx"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/state"
)

var _ = Describe("State Machine", func() {

	Context("when new Transitions are sent", func() {

		testCases := generateTestCases()
		for _, t := range testCases {
			t := t

			action := "<nil>"
			if t.finalAction != nil {
				action = reflect.TypeOf(t.finalAction).Name()
			}

			Context(fmt.Sprintf("when state machine begins with state - %s", reflect.TypeOf(t.startingState).Name()), func() {
				It(fmt.Sprintf("should eventually return action - %s", action), func() {
					pool := tx.FIFOPool(100)
					signer, err := ecdsa.NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
					shard := shard.Shard{
						Hash:        sig.Hash{},
						BlockHeader: sig.Hash{},
						BlockHeight: 0,
						Signatories: sig.Signatories{signer.Signatory()},
					}
					stateMachine := NewMachine(t.startingState, block.NewPolkaBuilder(), block.NewCommitBuilder(), signer, shard, pool, t.consensusThreshold)
					var action Action
					for _, transition := range t.transitions {
						if t.shouldPanic {
							Expect(func() { stateMachine.Transition(transition) }).To(Panic())
							return
						}
						action = stateMachine.Transition(transition)
					}
					if t.finalAction == nil {
						Expect(action).To(BeNil())
					} else {
						Expect(reflect.TypeOf(action).Name()).To(Equal(reflect.TypeOf(t.finalAction).Name()))
					}

				})
			})

		}
	})
})

type TestCase struct {
	consensusThreshold int

	startingState State
	finalAction   Action

	shouldPanic bool

	transitions []Transition
}

func generateTestCases() []TestCase {
	genesis := block.Genesis()
	signer, err := ecdsa.NewFromRandom()
	if err != nil {
		panic(fmt.Sprintf("error generating random SignerVerifier: %v", err))
	}

	return []TestCase{
		// (WaitForProposed) -> Proposed -> PreVoted (sig 1) -> PreCommitted (sig 1) -> PreCommitted (sig 2)
		{
			consensusThreshold: 2,

			startingState: WaitingForPropose{},
			finalAction:   Commit{},

			transitions: []Transition{
				Proposed{
					SignedPropose: block.SignedPropose{
						Propose: block.Propose{
							Block: genesis,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &genesis,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &genesis,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &genesis,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &genesis,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},

		// Invalid state
		{
			consensusThreshold: 1,

			startingState: InvalidState{},
			finalAction:   nil,

			shouldPanic: true,

			transitions: []Transition{
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 0,
								},
							},
							Round: 0,
						},
					},
				},
			},
		},

		// (WaitForPolka) -> Proposed -> PreVoted -> PreCommitted -> PreCommitted
		{
			consensusThreshold: 2,

			startingState: WaitingForPolka{},
			finalAction:   Commit{},

			transitions: []Transition{
				Proposed{
					SignedPropose: block.SignedPropose{
						Propose: block.Propose{
							Block: block.SignedBlock{
								Block: block.Block{},
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 0,
								},
							},
							Round: 0,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &block.SignedBlock{
									Block: block.Block{
										Height: 0,
									},
								},
								Round: 0,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &block.SignedBlock{
									Block: block.Block{
										Height: 0,
									},
								},
								Round: 0,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},

		// (WaitForCommit) -> Proposed -> PreVoted (sig 1) -> PreCommitted (sig 1) -> PreCommitted (sig 2)
		{
			consensusThreshold: 2,

			startingState: WaitingForCommit{},
			finalAction:   Commit{},

			transitions: []Transition{
				Proposed{
					SignedPropose: block.SignedPropose{
						Propose: block.Propose{
							Block: block.SignedBlock{
								Block: block.Block{
									Height: 0,
								},
							},
							Round: 0,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 0,
								},
							},
							Round: 0,
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &block.SignedBlock{
									Block: block.Block{
										Height: 0,
									},
								},
								Round: 0,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &block.SignedBlock{
									Block: block.Block{
										Height: 0,
									},
								},
								Round: 0,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},

		// Invalid transition
		{
			consensusThreshold: 1,

			startingState: WaitingForPropose{},
			finalAction:   nil,

			shouldPanic: true,

			transitions: []Transition{testutils.InvalidTransition{}},
		},

		// (WaitForPropose, TimedOut)
		{
			consensusThreshold: 1,

			startingState: WaitingForPropose{},
			finalAction: PreVote{
				PreVote: block.PreVote{},
			},

			transitions: []Transition{TimedOut{}},
		},

		// (WaitForPropose, Proposed) state.Round != block.Round
		{
			consensusThreshold: 1,

			startingState: WaitingForPropose{},
			finalAction:   PreVote{},

			transitions: []Transition{
				Proposed{
					SignedPropose: block.SignedPropose{
						Propose: block.Propose{
							Block: block.SignedBlock{
								Block: block.Block{},
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},

		// (WaitForPropose, Proposed) state.Height != block.Height
		{
			consensusThreshold: 1,

			startingState: WaitingForPropose{},
			finalAction:   PreVote{},

			transitions: []Transition{
				Proposed{
					SignedPropose: block.SignedPropose{
						Propose: block.Propose{
							Block: block.SignedBlock{
								Block: block.Block{},
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},

		// (WaitForPropose, Prevoted) state.Round != block.Round
		{
			consensusThreshold: 1,

			startingState: WaitingForPropose{},
			finalAction:   nil,

			transitions: []Transition{
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 0,
								},
							},
							Round:  0,
							Height: 0,
						},
					},
				},
			},
		},

		// (WaitForPropose, Prevoted) state.Height != block.Height
		{
			consensusThreshold: 1,

			startingState: WaitingForPropose{},
			finalAction:   nil,

			transitions: []Transition{
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 1,
								},
							},
							Round:  0,
							Height: 1,
						},
					},
				},
			},
		},

		// (WaitForPropose, PreCommitted) state.Round != block.Round
		{
			consensusThreshold: 1,

			startingState: WaitingForPropose{},
			finalAction:   Commit{},

			transitions: []Transition{
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &block.SignedBlock{
									Block: block.Block{
										Height: 0,
									},
								},
								Height: 0,
								Round:  0,
							},
						},
					},
				},
			},
		},

		// (WaitForCommit, PreCommitted) state.Round > polka.Round
		{
			consensusThreshold: 1,

			startingState: WaitingForCommit{},
			finalAction:   nil,

			transitions: []Transition{
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block:  nil,
								Height: 1,
								Round:  0,
							},
						},
						Signatory: testutils.RandomSignatory(),
						Signature: testutils.RandomSignature(),
					},
				},
			},
		},

		// (WaitForCommit, PreCommitted) PreCommits with same signatures
		{
			consensusThreshold: 2,

			startingState: WaitingForCommit{},
			finalAction:   nil,

			transitions: []Transition{
				PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(block.SignedBlock{}, signer, []sig.SignerVerifier{signer, signer}),
				},
				PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(block.SignedBlock{}, signer, []sig.SignerVerifier{signer, signer}),
				},
			},
		},

		// (WaitForPolka, Prevoted) PreVotes with same signatures
		{
			consensusThreshold: 2,

			startingState: WaitingForPolka{},
			finalAction:   nil,

			transitions: []Transition{
				PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(block.SignedBlock{}, signer),
				},
				PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(block.SignedBlock{}, signer),
				},
			},
		},
	}
}

type InvalidState struct{}

func (state InvalidState) IsState() {}
