package replica

import (
	"time"

	"github.com/renproject/hyperdrive/block"
)

type State interface {
	IsState()
}

type WaitingForPropose struct {
	Round  block.Round
	Height block.Height
}

func (waitingForPropose WaitingForPropose) IsState() {
}

type WaitingForPolka struct {
	Round  block.Round
	Height block.Height
}

func (waitingForPolka WaitingForPolka) IsState() {
}

type WaitingForCommit struct {
	Polka block.Polka
}

func (waitingForCommit WaitingForCommit) IsState() {
}

type Dispatcher interface {
	Dispatch(action Action)
}

type StateMachine interface {
	Transition(transition Transition)
}

type stateMachine struct {
	state      State
	dispatcher Dispatcher

	polkaBuilder       block.PolkaBuilder
	commitBuilder      block.CommitBuilder
	blockchain         block.Blockchain
	consensusThreshold int64
}

func NewStateMachine(state State, dispatcher Dispatcher, polkaBuilder block.PolkaBuilder, commitBuilder block.CommitBuilder, blockchain block.Blockchain, consensusThreshold int64) StateMachine {
	return &stateMachine{
		state:              state,
		dispatcher:         dispatcher,
		polkaBuilder:       polkaBuilder,
		commitBuilder:      commitBuilder,
		blockchain:         blockchain,
		consensusThreshold: consensusThreshold,
	}
}

func (stateMachine *stateMachine) Transition(transition Transition) {
	switch state := stateMachine.state.(type) {
	case WaitingForPropose:
		stateMachine.transitionFromWaitingForPropose(state, transition)

	case WaitingForPolka:
		stateMachine.transitionFromWaitingForPolka(state, transition)

	case WaitingForCommit:
		stateMachine.transitionFromWaitingForCommit(state, transition)
	}
}

func (stateMachine *stateMachine) transitionFromWaitingForPropose(state WaitingForPropose, transition Transition) {
	switch transition := transition.(type) {
	case TimedOut:
		stateMachine.dispatcher.Dispatch(PreVote{
			PreVote: block.PreVote{
				Block:  nil,
				Round:  state.Round,
				Height: stateMachine.blockchain.Height() + 1,
			},
		})
		stateMachine.state = WaitingForPolka{
			Round: state.Round,
		}

	case Proposed:
		if transition.Block.Round != state.Round {
			return
		}
		if transition.Block.Height != state.Height {
			return
		}
		if transition.Block.Time.After(time.Now()) {
			return
		}
		stateMachine.dispatcher.Dispatch(PreVote{
			PreVote: block.PreVote{
				Block:  &transition.Block,
				Round:  transition.Block.Round,
				Height: transition.Block.Height,
			},
		})
		stateMachine.state = WaitingForPolka{
			Round: state.Round,
		}

	case PreVoted:
		if transition.Round != state.Round {
			return
		}
		if transition.Height != state.Height {
			return
		}
		stateMachine.polkaBuilder.Insert(transition.SignedPreVote)
		if polka, ok := stateMachine.polkaBuilder.Polka(stateMachine.consensusThreshold); ok {
			stateMachine.dispatcher.Dispatch(PreCommit{
				PreCommit: block.PreCommit{
					Polka: polka,
				},
			})
			stateMachine.state = WaitingForCommit{
				Polka: polka,
			}
		}

	case PreCommitted:
		if transition.Polka.Round != state.Round {
			return
		}
		if transition.Polka.Height != state.Height {
			return
		}
		stateMachine.commitBuilder.Insert(transition.SignedPreCommit)
		if commit, ok := stateMachine.commitBuilder.Commit(stateMachine.consensusThreshold); ok {
			if commit.Polka.Block == nil {
				stateMachine.state = WaitingForPropose{
					Round:  state.Round + 1,
					Height: state.Height,
				}
				return
			}
			stateMachine.blockchain.Extend(*commit.Polka.Block)
			stateMachine.state = WaitingForPropose{
				Round:  state.Round,
				Height: state.Height + 1,
			}
			return
		}
		stateMachine.dispatcher.Dispatch(PreCommit{
			PreCommit: block.PreCommit{
				Polka: transition.Polka,
			},
		})
		stateMachine.state = WaitingForCommit{
			Polka: transition.Polka,
		}
	}
}

func (stateMachine *stateMachine) transitionFromWaitingForPolka(state WaitingForPolka, transition Transition) {
	switch transition := transition.(type) {

	case PreVoted:
		if transition.Round != state.Round {
			return
		}
		if transition.Height != state.Height {
			return
		}
		stateMachine.polkaBuilder.Insert(transition.SignedPreVote)
		if polka, ok := stateMachine.polkaBuilder.Polka(stateMachine.consensusThreshold); ok {
			stateMachine.dispatcher.Dispatch(PreCommit{
				PreCommit: block.PreCommit{
					Polka: polka,
				},
			})
			stateMachine.state = WaitingForCommit{
				Polka: polka,
			}
			return
		}

	case PreCommitted:
		if transition.Polka.Round != state.Round {
			return
		}
		if transition.Polka.Height != state.Height {
			return
		}
		stateMachine.commitBuilder.Insert(transition.SignedPreCommit)
		if commit, ok := stateMachine.commitBuilder.Commit(stateMachine.consensusThreshold); ok {
			if commit.Polka.Block == nil {
				stateMachine.state = WaitingForPropose{
					Round:  state.Round + 1,
					Height: state.Height,
				}
				return
			}
			stateMachine.blockchain.Extend(*commit.Polka.Block)
			stateMachine.state = WaitingForPropose{
				Round:  state.Round,
				Height: state.Height + 1,
			}
			return
		}
		stateMachine.dispatcher.Dispatch(PreCommit{
			PreCommit: block.PreCommit{
				Polka: transition.Polka,
			},
		})
		stateMachine.state = WaitingForCommit{
			Polka: transition.Polka,
		}
	}
}

func (stateMachine *stateMachine) transitionFromWaitingForCommit(state WaitingForCommit, transition Transition) {
	switch transition := transition.(type) {

	case PreCommitted:
		if transition.Polka.Round != state.Polka.Round {
			return
		}
		if transition.Polka.Height != state.Polka.Height {
			return
		}
		stateMachine.commitBuilder.Insert(transition.SignedPreCommit)
		if commit, ok := stateMachine.commitBuilder.Commit(stateMachine.consensusThreshold); ok {
			if commit.Polka.Block == nil {
				stateMachine.state = WaitingForPropose{
					Round:  state.Polka.Round + 1,
					Height: state.Polka.Height,
				}
				return
			}
			stateMachine.blockchain.Extend(*commit.Polka.Block)
			stateMachine.state = WaitingForPropose{
				Round:  state.Polka.Round,
				Height: state.Polka.Height + 1,
			}
			return
		}

	}
}