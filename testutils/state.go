package testutils

import (
	"github.com/renproject/hyperdrive/block"
)

type InvalidState struct{}

func (state InvalidState) Round() block.Round {
	return 0
}

func (state InvalidState) Height() block.Height {
	return 0
}
