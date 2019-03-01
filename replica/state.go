package replica

import (
	"github.com/renproject/hyperdrive/block"
)

type State interface {
	Round() block.Round
	Height() block.Height
}

type WaitingForPropose struct {
	round  block.Round
	height block.Height
}

func WaitForPropose(round block.Round, height block.Height) State {
	return WaitingForPropose{
		round:  round,
		height: height,
	}
}

func (waitingForPropose WaitingForPropose) Round() block.Round {
	return waitingForPropose.round
}

func (waitingForPropose WaitingForPropose) Height() block.Height {
	return waitingForPropose.height
}

type WaitingForPolka struct {
	round  block.Round
	height block.Height
}

func WaitForPolka(round block.Round, height block.Height) State {
	return WaitingForPolka{
		round:  round,
		height: height,
	}
}

func (waitingForPolka WaitingForPolka) Round() block.Round {
	return waitingForPolka.round
}

func (waitingForPolka WaitingForPolka) Height() block.Height {
	return waitingForPolka.height
}

type WaitingForCommit struct {
	polka block.Polka
}

func WaitForCommit(polka block.Polka) State {
	return WaitingForCommit{
		polka: polka,
	}
}

func (waitingForCommit WaitingForCommit) Round() block.Round {
	return waitingForCommit.polka.Round
}

func (waitingForCommit WaitingForCommit) Height() block.Height {
	return waitingForCommit.polka.Height
}
