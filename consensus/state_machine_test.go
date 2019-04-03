package consensus_test

import (
	"crypto/rand"
	"fmt"
	"reflect"

	"github.com/renproject/hyperdrive/v1/block"
	"github.com/renproject/hyperdrive/v1/sig"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/v1/consensus"
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
					stateMachine := NewStateMachine(block.PolkaBuilder{}, block.CommitBuilder{}, t.consensusThreshold)
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
						Signatory: randomSignatory(),
						Signature: randomSignature(),
					},
				},
				PreVoted{
					block.SignedPreVote{
						PreVote: block.PreVote{
							Block: &genesis,
						},
						Signatory: randomSignatory(),
						Signature: randomSignature(),
					},
				},
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &genesis,
							},
						},
						Signatory: randomSignatory(),
						Signature: randomSignature(),
					},
				},
				PreCommitted{
					block.SignedPreCommit{
						PreCommit: block.PreCommit{
							Polka: block.Polka{
								Block: &genesis,
							},
						},
						Signatory: randomSignatory(),
						Signature: randomSignature(),
					},
				},
			},
		},

		// Invalid state
		{
			consensusThreshold: 1,

			startingState: InvalidState{height: 0, round: 0},
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
						Signatory: randomSignatory(),
						Signature: randomSignature(),
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
						Signatory: randomSignatory(),
						Signature: randomSignature(),
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
						Signatory: randomSignatory(),
						Signature: randomSignature(),
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
						Signatory: randomSignatory(),
						Signature: randomSignature(),
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
						Signatory: randomSignatory(),
						Signature: randomSignature(),
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
						Signatory: randomSignatory(),
						Signature: randomSignature(),
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

			transitions: []Transition{InvalidTransition{}},
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
						Signatory: randomSignatory(),
						Signature: randomSignature(),
					},
				},
			},
		},
	}
}

type InvalidState struct {
	round  block.Round
	height block.Height
}

func (state InvalidState) Round() block.Round {
	return state.round
}

func (state InvalidState) Height() block.Height {
	return state.height
}

type InvalidTransition struct {
}

func (transition InvalidTransition) IsTransition() {}

func randomSignatory() sig.Signatory {
	key := make([]byte, 20)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	signatory := sig.Signatory{}
	copy(signatory[:], key[:])

	return signatory
}

func randomSignature() sig.Signature {
	key := make([]byte, 65)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	sign := sig.Signature{}
	copy(sign[:], key[:])

	return sign
}
