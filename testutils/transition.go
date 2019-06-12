package testutils

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
)

type InvalidTransition struct {
}

func (transition InvalidTransition) IsTransition() {}

func (transition InvalidTransition) Round() block.Round {
	return 0
}

func (transition InvalidTransition) Signer() sig.Signatory {
	return sig.Signatory{}
}
