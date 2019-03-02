package replica

import (
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/supervisor"
)

type Broadcaster interface {
	Send(to sig.Signatory, data interface{})
	Broadcast(data interface{})
}

type Replica interface {
}

type replica struct {
	state        State
	stateMachine StateMachine
}

func New(state State, stateMachine StateMachine) Replica {
	return &replica{
		state:        state,
		stateMachine: stateMachine,
	}
}

func Run(ctx supervisor.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
