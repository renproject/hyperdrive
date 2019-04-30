package state_test

import (
	"fmt"
	"reflect"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/state"
)

var _ = Describe("State Machine", func() {

	Context("when new Transitions are sent", func() {

		testCases := generateTestCases()
		for _, t := range testCases {
			t := t

			stateStr := "<nil>"
			if t.finalState != nil {
				stateStr = reflect.TypeOf(t.finalState).Name()
			}

			Context(fmt.Sprintf("when state machine begins with state - %s", reflect.TypeOf(t.startingState).Name()), func() {
				It(fmt.Sprintf("should eventually arrive at state %s", stateStr), func() {
					stateMachine := NewMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), t.consensusThreshold)
					state := t.startingState
					var action Action
					for _, transition := range t.transitions {
						state, action = stateMachine.Transition(state, transition)
					}
					if t.finalState == nil {
						Expect(state).To(BeNil())
					} else {
						Expect(state).To(Equal(t.finalState))
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
	finalState    State
	finalAction   Action

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

			startingState: WaitForPropose(0, 0),
			finalState:    WaitForPropose(0, 1),
			finalAction:   Commit{},

			transitions: []Transition{
				Proposed{
					SignedBlock: genesis,
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
			finalState:    nil,
			finalAction:   nil,

			transitions: []Transition{
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 0,
									Round:  0,
								},
							},
						},
					},
				},
			},
		},

		// (WaitForPolka) -> Proposed -> PreVoted -> PreCommitted -> PreCommitted
		{
			consensusThreshold: 2,

			startingState: WaitForPolka(0, 0),
			finalState:    WaitForPropose(0, 1),
			finalAction:   Commit{},

			transitions: []Transition{
				Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{},
					},
				},
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 0,
									Round:  0,
								},
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
										Round:  0,
									},
								},
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
										Round:  0,
									},
								},
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

			startingState: WaitForCommit(block.Polka{
				Block: &block.SignedBlock{
					Block: block.Block{
						Height: 0,
						Round:  0,
					},
				},
			}),
			finalState:  WaitForPropose(0, 1),
			finalAction: Commit{},

			transitions: []Transition{
				Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{
							Height: 0,
							Round:  0,
						},
					},
				},
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 0,
									Round:  0,
								},
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
										Round:  0,
									},
								},
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
										Round:  0,
									},
								},
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

			startingState: WaitForPropose(0, 0),
			finalState:    WaitForPropose(0, 0),
			finalAction:   nil,

			transitions: []Transition{testutils.InvalidTransition{}},
		},

		// (WaitForPropose, TimedOut)
		{
			consensusThreshold: 1,

			startingState: WaitForPropose(0, 0),
			finalState:    WaitForPolka(0, 0),
			finalAction: PreVote{
				PreVote: block.PreVote{},
			},

			transitions: []Transition{TimedOut{}},
		},

		// (WaitForPropose, Proposed) state.Round != block.Round
		{
			consensusThreshold: 1,

			startingState: WaitForPropose(1, 0),
			finalState:    WaitForPropose(1, 0),
			finalAction:   nil,

			transitions: []Transition{
				Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{},
					},
				},
			},
		},

		// (WaitForPropose, Proposed) state.Height != block.Height
		{
			consensusThreshold: 1,

			startingState: WaitForPropose(0, 1),
			finalState:    WaitForPropose(0, 1),
			finalAction:   nil,

			transitions: []Transition{
				Proposed{
					SignedBlock: block.SignedBlock{
						Block: block.Block{},
					},
				},
			},
		},

		// (WaitForPropose, Prevoted) state.Round != block.Round
		{
			consensusThreshold: 1,

			startingState: WaitForPropose(1, 0),
			finalState:    WaitForPropose(1, 0),
			finalAction:   nil,

			transitions: []Transition{
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 0,
									Round:  0,
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

			startingState: WaitForPropose(0, 0),
			finalState:    WaitForPropose(0, 0),
			finalAction:   nil,

			transitions: []Transition{
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &block.SignedBlock{
								Block: block.Block{
									Height: 1,
									Round:  0,
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

			startingState: WaitForPropose(0, 1),
			finalState:    WaitForPropose(0, 1),
			finalAction:   nil,

			transitions: []Transition{
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &block.SignedBlock{
									Block: block.Block{
										Height: 0,
										Round:  0,
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

			startingState: WaitForCommit(block.Polka{
				Block:  nil,
				Height: 1,
				Round:  1,
			}),
			finalState: WaitForCommit(block.Polka{
				Block:  nil,
				Height: 1,
				Round:  1,
			}),
			finalAction: nil,

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

			startingState: WaitForCommit(block.Polka{
				Block:  nil,
				Height: 0,
				Round:  0,
			}),
			finalState:  WaitForCommit(testutils.GeneratePolkaWithSignatures(block.SignedBlock{}, []sig.SignerVerifier{signer, signer})),
			finalAction: nil,

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

			startingState: WaitForPolka(0, 0),
			finalState:    WaitForPolka(0, 0),
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

func (state InvalidState) Round() block.Round {
	return 0
}

func (state InvalidState) Height() block.Height {
	return 0
}
