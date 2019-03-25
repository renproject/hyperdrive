package consensus_test

import (
	"crypto/rand"
	"fmt"
	"reflect"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/consensus"
)

var _ = Describe("State Machine", func() {

	stateMachine := NewStateMachine(block.PolkaBuilder{}, block.CommitBuilder{}, 2)

	testCases := generateTestCases()

	for _, t := range testCases {
		t := t

		Context(fmt.Sprintf("when the state machine gets (%s, %s)", reflect.TypeOf(t.inputState).Name(), reflect.TypeOf(t.inputTransition).Name()), func() {
			It(fmt.Sprintf("should return (%s, %s)", t.expectedState, t.expectedAction), func() {
				state, action := stateMachine.Transition(t.inputState, t.inputTransition)
				t.validateResults(state, action)
			})
		})
	}
})

type TestCase struct {
	inputState      State
	inputTransition Transition

	expectedState   string
	expectedAction  string
	validateResults func(state State, action Action)
}

func generateTestCases() []TestCase {
	genesis := block.Genesis()

	return []TestCase{

		// Invalid state
		{
			inputState: InvalidState{height: 0, round: 0},
			inputTransition: PreVoted{
				block.SignedPreVote{
					PreVote: block.PreVote{
						Block: &block.Block{
							Height: 0,
							Round:  0,
						},
					},
				},
			},

			expectedState:  "nil",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				Expect(state).To(BeNil())
				Expect(action).To(BeNil())
			},
		},

		// Invalid transition
		{
			inputState:      WaitForPropose(0, 0),
			inputTransition: InvalidTransition{},

			expectedState:  "WaitingForPropose",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPolka")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPropose, TimedOut)
		{
			inputState:      WaitForPropose(0, 0),
			inputTransition: TimedOut{},

			expectedState:  "WaitingForPolka",
			expectedAction: "PreVote",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPolka)
				Expect(ok).To(Equal(true), "state should have been WaitingForPolka")
				prevote, ok := action.(PreVote)
				Expect(ok).To(Equal(true), "action should have been PreVote")
				Expect(prevote.PreVote.Round).To(Equal(genesis.Round))
				Expect(prevote.PreVote.Height).To(Equal(genesis.Height))
				Expect(prevote.PreVote.Block).To(BeNil())
			},
		},

		// (WaitForPropose, Proposed)
		{
			inputState:      WaitForPropose(0, 0),
			inputTransition: Proposed{block.Block{}},

			expectedState:  "WaitingForPolka",
			expectedAction: "PreVote",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPolka)
				Expect(ok).To(Equal(true), "state should have been WaitingForPolka")
				prevote, ok := action.(PreVote)
				Expect(ok).To(Equal(true), "action should have been PreVote")
				Expect(prevote.PreVote.Round).To(Equal(genesis.Round))
				Expect(prevote.PreVote.Height).To(Equal(genesis.Height))
				verifyBlock(prevote.PreVote.Block, &genesis)
			},
		},

		// (WaitForPropose, PreVoted)
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: PreVoted{
				block.SignedPreVote{
					PreVote: block.PreVote{
						Block: &genesis,
					},
					Signatory: randomSignatory(),
					Signature: randomSignature(),
				},
			},

			expectedState:  "WaitingForPolka",
			expectedAction: "PreVote",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPolka)
				Expect(ok).To(Equal(true), "state should have been WaitingForPolka")
				prevote, ok := action.(PreVote)
				Expect(ok).To(Equal(true), "action should have been PreVote")
				Expect(prevote.PreVote.Round).To(Equal(genesis.Round))
				Expect(prevote.PreVote.Height).To(Equal(genesis.Height))
				verifyBlock(prevote.PreVote.Block, &genesis)
			},
		},

		// (WaitForPropose, PreVoted)
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: PreVoted{
				block.SignedPreVote{
					PreVote: block.PreVote{
						Block: &genesis,
					},
					Signatory: randomSignatory(),
					Signature: randomSignature(),
				},
			},

			expectedState:  "WaitingForCommit",
			expectedAction: "PreCommit",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForCommit)
				fmt.Println(reflect.TypeOf(state).Name())
				Expect(ok).To(Equal(true), "state should have been WaitingForCommit")
				precommit, ok := action.(PreCommit)
				Expect(ok).To(Equal(true), "action should have been PreCommit")
				Expect(precommit.PreCommit.Polka.Round).To(Equal(genesis.Round))
				Expect(precommit.PreCommit.Polka.Height).To(Equal(genesis.Height))
				verifyBlock(precommit.PreCommit.Polka.Block, &genesis)
			},
		},

		// (WaitForPropose, PreCommitted)
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: PreCommitted{
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

			expectedState:  "WaitingForCommit",
			expectedAction: "PreCommit",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForCommit)
				Expect(ok).To(Equal(true), "state should have been WaitingForCommit")
				precommit, ok := action.(PreCommit)
				Expect(ok).To(Equal(true), "action should have been PreCommit")
				Expect(precommit.PreCommit.Polka.Round).To(Equal(genesis.Round))
				Expect(precommit.PreCommit.Polka.Height).To(Equal(genesis.Height))
				verifyBlock(precommit.PreCommit.Polka.Block, &genesis)
			},
		},

		// (WaitForPropose, PreCommitted)
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: PreCommitted{
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

			expectedState:  "WaitingForPropose",
			expectedAction: "Commit",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				commit, ok := action.(Commit)
				Expect(ok).To(Equal(true), "action should have been Commit")
				Expect(commit.Commit.Polka.Round).To(Equal(genesis.Round))
				Expect(commit.Commit.Polka.Height).To(Equal(genesis.Height))
				verifyBlock(commit.Commit.Polka.Block, &genesis)
			},
		},

		// (WaitForPropose, Proposed) state.Round != block.Round
		{
			inputState:      WaitForPropose(1, 0),
			inputTransition: Proposed{block.Block{}},

			expectedState:  "WaitingForPropose",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPropose, Proposed) state.Height != block.Height
		{
			inputState:      WaitForPropose(0, 1),
			inputTransition: Proposed{block.Block{}},

			expectedState:  "WaitingForPropose",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPropose, Proposed) block.Time > time.Now()
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: Proposed{block.Block{
				Time: time.Now().Add(10 * time.Minute),
			}},

			expectedState:  "WaitingForPropose",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPropose, Prevoted) state.Round != block.Round
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: PreVoted{
				block.SignedPreVote{
					PreVote: block.PreVote{
						Block: &block.Block{
							Height: 0,
							Round:  1,
						},
						Round:  1,
						Height: 0,
					},
				},
			},

			expectedState:  "WaitingForPropose",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPropose, Prevoted) state.Height != block.Height
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: PreVoted{
				block.SignedPreVote{
					PreVote: block.PreVote{
						Block: &block.Block{
							Height: 1,
							Round:  0,
						},
						Round:  0,
						Height: 1,
					},
				},
			},

			expectedState:  "WaitingForPropose",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPropose, PreCommitted) state.Round != block.Round
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: PreCommitted{
				block.SignedPreCommit{
					PreCommit: block.PreCommit{
						Polka: block.Polka{
							Block: &block.Block{
								Height: 0,
								Round:  0,
							},
							Height: 0,
							Round:  1,
						},
					},
				},
			},

			expectedState:  "WaitingForPropose",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPropose, PreCommitted) state.Height != block.Height
		{
			inputState: WaitForPropose(0, 0),
			inputTransition: PreCommitted{
				block.SignedPreCommit{
					PreCommit: block.PreCommit{
						Polka: block.Polka{
							Block: &block.Block{
								Height: 0,
								Round:  0,
							},
							Height: 1,
							Round:  0,
						},
					},
					Signatory: randomSignatory(),
					Signature: randomSignature(),
				},
			},

			expectedState:  "WaitingForPropose",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPolka, Proposed)
		{
			inputState:      WaitForPolka(0, 0),
			inputTransition: Proposed{block.Block{}},

			expectedState:  "WaitingForPolka",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPolka)
				Expect(ok).To(Equal(true), "state should have been WaitingForPolka")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForPolka, PreVoted)
		{
			inputState: WaitForPolka(0, 0),
			inputTransition: PreVoted{
				block.SignedPreVote{
					PreVote: block.PreVote{
						Block: &block.Block{
							Height: 0,
							Round:  0,
						},
					},
					Signatory: randomSignatory(),
					Signature: randomSignature(),
				},
			},

			expectedState:  "WaitingForCommit",
			expectedAction: "PreCommit",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForCommit)
				Expect(ok).To(Equal(true), "state should have been WaitingForCommit")
				precommit, ok := action.(PreCommit)
				Expect(ok).To(Equal(true), "action should have been PreCommit")
				Expect(precommit.PreCommit.Polka.Round).To(Equal(genesis.Round))
				Expect(precommit.PreCommit.Polka.Height).To(Equal(genesis.Height))
				verifyBlock(precommit.PreCommit.Polka.Block, &genesis)
			},
		},

		// (WaitForPolka, PreCommitted)
		{
			inputState: WaitForPolka(0, 0),
			inputTransition: PreCommitted{
				block.SignedPreCommit{
					PreCommit: block.PreCommit{
						Polka: block.Polka{
							Block: &block.Block{
								Height: 0,
								Round:  0,
							},
						},
					},
					Signatory: randomSignatory(),
					Signature: randomSignature(),
				},
			},

			expectedState:  "WaitingForPropose",
			expectedAction: "Commit",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				commit, ok := action.(Commit)
				Expect(ok).To(Equal(true), "action should have been Commit")
				Expect(commit.Commit.Polka.Round).To(Equal(genesis.Round))
				Expect(commit.Commit.Polka.Height).To(Equal(genesis.Height))
				verifyBlock(commit.Commit.Polka.Block, &genesis)
			},
		},

		// (WaitForCommit, Proposed)
		{
			inputState: WaitForCommit(block.Polka{
				Block: &block.Block{
					Height: 0,
					Round:  0,
				},
			}),
			inputTransition: Proposed{
				block.Block{
					Height: 0,
					Round:  0,
				}},

			expectedState:  "WaitingForCommit",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForCommit)
				Expect(ok).To(Equal(true), "state should have been WaitingForCommit")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForCommit, PreVoted)
		{
			inputState: WaitForCommit(block.Polka{
				Block: &block.Block{
					Height: 0,
					Round:  0,
				},
			}),
			inputTransition: PreVoted{
				block.SignedPreVote{
					PreVote: block.PreVote{
						Block: &block.Block{
							Height: 0,
							Round:  0,
						},
					},
					Signatory: randomSignatory(),
					Signature: randomSignature(),
				},
			},

			expectedState:  "WaitingForCommit",
			expectedAction: "nil",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForCommit)
				Expect(ok).To(Equal(true), "state should have been WaitingForCommit")
				Expect(action).To(BeNil())
			},
		},

		// (WaitForCommit, PreCommitted)
		{
			inputState: WaitForCommit(block.Polka{
				Block: &genesis,
			}),
			inputTransition: PreCommitted{
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

			expectedState:  "WaitingForPropose",
			expectedAction: "Commit",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				commit, ok := action.(Commit)
				Expect(ok).To(Equal(true), "action should have been Commit")
				Expect(commit.Commit.Polka.Round).To(Equal(genesis.Round))
				Expect(commit.Commit.Polka.Height).To(Equal(genesis.Height))
				verifyBlock(commit.Commit.Polka.Block, &genesis)
			},
		},

		// (WaitForCommit, PreCommitted)
		{
			inputState: WaitForCommit(block.Polka{
				Block:  nil,
				Height: 1,
				Round:  1,
			}),
			inputTransition: PreCommitted{
				block.SignedPreCommit{
					PreCommit: block.PreCommit{
						Polka: block.Polka{
							Block:  nil,
							Height: 1,
							Round:  1,
						},
					},
					Signatory: randomSignatory(),
					Signature: randomSignature(),
				},
			},

			expectedState:  "WaitingForPropose",
			expectedAction: "Commit",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				commit, ok := action.(Commit)
				Expect(ok).To(Equal(true), "action should have been Commit")
				Expect(commit.Commit.Polka.Round).NotTo(Equal(1))
				Expect(commit.Commit.Polka.Height).NotTo(Equal(1))
			},
		},

		// (WaitForCommit, PreCommitted)
		{
			inputState: WaitForCommit(block.Polka{
				Block:  nil,
				Height: 1,
				Round:  1,
			}),
			inputTransition: PreCommitted{
				block.SignedPreCommit{
					PreCommit: block.PreCommit{
						Polka: block.Polka{
							Block:  nil,
							Height: 1,
							Round:  1,
						},
					},
					Signatory: randomSignatory(),
					Signature: randomSignature(),
				},
			},

			expectedState:  "WaitingForPropose",
			expectedAction: "Commit",
			validateResults: func(state State, action Action) {
				_, ok := state.(WaitingForPropose)
				Expect(ok).To(Equal(true), "state should have been WaitingForPropose")
				Expect(action).To(BeNil())
			},
		},
	}
}

func verifyBlock(res *block.Block, expected *block.Block) {
	Expect(res.Round).To(Equal(expected.Round))
	Expect(res.Height).To(Equal(expected.Height))
	Expect(res.Header).To(Equal(expected.Header))
	Expect(res.ParentHeader).To(Equal(expected.ParentHeader))
	Expect(res.Signature).To(Equal(expected.Signature))
	Expect(res.Signatory).To(Equal(expected.Signatory))
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
