package testutils

import (
	"github.com/renproject/hyperdrive/consensus"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/sig"
)

type MockDispatcher struct {
}

func NewMockDispatcher() replica.Dispatcher {
	return &MockDispatcher{}
}

func (dispatcher *MockDispatcher) Dispatch(shardHash sig.Hash, action consensus.Action) {}
