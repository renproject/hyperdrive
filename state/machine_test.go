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
		signer, err := ecdsa.NewFromRandom()
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, t := range generateTestCases(signer) {
			t := t

			action := "<nil>"
			if t.finalAction != nil {
				action = reflect.TypeOf(t.finalAction).Name()
			}

			Context(fmt.Sprintf("when state machine begins with state - %s", reflect.TypeOf(t.startingState).Name()), func() {
				It(fmt.Sprintf("should eventually return action - %s", action), func() {
					pool := tx.FIFOPool(100)

					shard := shard.Shard{
						Hash:        sig.Hash{},
						BlockHeader: sig.Hash{},
						BlockHeight: 0,
						Signatories: sig.Signatories{signer.Signatory(), testutils.RandomSignatory()},
					}
					stateMachine := NewMachine(t.startingState, block.NewPolkaBuilder(), block.NewCommitBuilder(), signer, shard, pool, nil)
					var action Action
					for _, transition := range t.transitions {
						if t.shouldPanic {
							Expect(func() { stateMachine.Transition(transition) }).To(Panic())
							return
						}
						action = stateMachine.Transition(transition)
					}

					Expect(reflect.TypeOf(stateMachine.State()).Name()).To(Equal(reflect.TypeOf(t.finalState).Name()))
					Expect(stateMachine.Round()).To(Equal(t.finalRound))
					Expect(stateMachine.Height()).To(Equal(t.finalHeight))

					if t.finalAction == nil {
						Expect(action).To(BeNil())
					} else {
						Expect(action).NotTo(BeNil())
						Expect(reflect.TypeOf(action).Name()).To(Equal(reflect.TypeOf(t.finalAction).Name()))
					}

				})
			})

		}
	})
})

type TestCase struct {
	startingState State

	transitions []Transition

	finalState  State
	finalAction Action
	finalRound  block.Round
	finalHeight block.Height
	shouldPanic bool
}

func generateTestCases(signer sig.SignerVerifier) []TestCase {
	genesis := block.Genesis()
	propose := block.Propose{
		Block: block.SignedBlock{
			Block: block.Block{},
		},
		ValidRound: -1,
	}

	signedPropose, err := propose.Sign(signer)
	if err != nil {
		fmt.Println(err)
	}

	return []TestCase{
		// (WaitForProposed) -> Proposed -> PreVoted (sig 1) -> PreCommitted (sig 1) -> PreCommitted (sig 2)
		{
			startingState: WaitingForPropose{},
			finalState:    WaitingForPropose{},
			finalAction:   Propose{},
			finalRound:    0,
			finalHeight:   1,

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

			startingState: InvalidState{},
			finalState:    WaitingForPropose{},
			finalAction:   nil,
			finalRound:    0,
			finalHeight:   0,

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

			startingState: WaitingForPolka{},
			finalState:    WaitingForPropose{},
			finalAction:   Propose{},
			finalRound:    0,
			finalHeight:   1,

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

		// (WaitForCommit) 	-> Proposed(Round:1)
		//					-> PreVoted(Round:1)  (sig 1)
		//  				-> PreCommitted(Round:1)  (sig 1)
		// 					-> PreCommitted(Round:1)  (sig 2)
		{

			startingState: WaitingForCommit{},
			finalState:    WaitingForPropose{},
			finalAction:   Propose{},
			finalRound:    0,
			finalHeight:   1,

			transitions: []Transition{
				Proposed{
					SignedPropose: block.SignedPropose{
						Propose: block.Propose{
							Block: block.SignedBlock{
								Block: block.Block{
									Height: 0,
								},
							},
							Round: 1,
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
							Round: 1,
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
								Round: 1,
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
								Round: 1,
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

			startingState: WaitingForCommit{},
			finalState:    WaitingForPropose{},
			finalAction:   Propose{},
			finalRound:    0,
			finalHeight:   1,

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

			startingState: WaitingForPropose{},
			finalAction:   nil,
			finalState:    WaitingForPropose{},

			shouldPanic: true,

			transitions: []Transition{testutils.InvalidTransition{}},
		},

		// (WaitForCommit -> WaitForPropose -> Ticked -> Ticked -> (ProposeTimedOut) -> SignedPreVote)
		{
			startingState: WaitingForCommit{},
			finalState:    WaitingForPolka{},
			finalAction:   SignedPreVote{},
			finalRound:    0,
			finalHeight:   1,

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
				Ticked{},
				Ticked{},
			},
		},

		// (WaitForPropose, Proposed) state.Round != block.Round
		{
			startingState: WaitingForPropose{},
			finalState:    WaitingForPolka{},
			finalAction:   SignedPreVote{},
			finalRound:    0,
			finalHeight:   0,

			transitions: []Transition{
				Proposed{
					SignedPropose: signedPropose,
				},
			},
		},

		// (WaitForPropose, Proposed) invalid leader
		{

			startingState: WaitingForPropose{},
			finalState:    WaitingForPropose{},
			finalAction:   nil,
			finalRound:    0,
			finalHeight:   0,

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

		// (WaitForPropose, Prevoted) state.Height != block.Height
		{

			startingState: WaitingForPropose{},

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
							Height: 0,
						},
					},
				},
			},

			shouldPanic: true,

			finalState:  WaitingForPropose{},
			finalAction: nil,
			finalRound:  0,
			finalHeight: 0,
		},

		// (WaitForPropose, Prevoted) state.Height != block.Height
		{

			startingState: WaitingForPropose{},

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

			finalState:  WaitingForPropose{},
			finalAction: nil,
			finalRound:  0,
			finalHeight: 0,
		},

		// (WaitForPropose, PreCommitted) state.Height != block.Height
		{
			startingState: WaitingForPropose{},

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
								Height: 1,
								Round:  0,
							},
						},
					},
				},
			},

			shouldPanic: true,

			finalState:  WaitingForPropose{},
			finalAction: nil,
			finalRound:  0,
			finalHeight: 0,
		},

		// (WaitForCommit, PreCommitted) state.Height > polka.Height
		{
			startingState: WaitingForCommit{},

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

			finalState:  WaitingForCommit{},
			finalAction: nil,
			finalRound:  0,
			finalHeight: 0,
		},

		// (WaitForCommit, PreCommitted) PreCommits with same signatures
		{

			startingState: WaitingForCommit{},
			finalState:    WaitingForCommit{},
			finalAction:   nil,
			finalRound:    0,
			finalHeight:   0,

			transitions: []Transition{
				PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(block.SignedBlock{}, signer, []sig.SignerVerifier{signer, signer}),
				},
				PreCommitted{
					SignedPreCommit: testutils.GenerateSignedPreCommit(block.SignedBlock{}, signer, []sig.SignerVerifier{signer, signer}),
				},
			},
		},

		// (WaitForPolka, Prevoted) PreVotes with same signatures should not change state
		{

			startingState: WaitingForPolka{},
			finalState:    WaitingForPolka{},
			finalAction:   nil,
			finalRound:    0,
			finalHeight:   0,

			transitions: []Transition{
				PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(block.SignedBlock{}, signer),
				},
				PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(block.SignedBlock{}, signer),
				},
			},
		},

		// (WaitForPolka, Prevoted) PreVotes with different signatures should change state to WaitingForCommit
		{

			startingState: WaitingForPolka{},
			finalState:    WaitingForCommit{},
			finalAction:   SignedPreCommit{},
			finalRound:    0,
			finalHeight:   0,

			transitions: []Transition{
				PreVoted{
					SignedPreVote: testutils.GenerateSignedPreVote(block.SignedBlock{}, signer),
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
			},
		},
	}
}

type InvalidState struct{}

func (state InvalidState) IsState() {}
