package consensus

import "github.com/renproject/hyperdrive/block"

type Action interface {
	IsAction()
}

type Propose struct {
	block.Block
}

func (Propose) IsAction() {
}

type PreVote struct {
	block.PreVote
}

func (PreVote) IsAction() {
}

type PreCommit struct {
	block.PreCommit
}

func (PreCommit) IsAction() {
}

type Commit struct {
	block.Commit
}

func (Commit) IsAction() {
}
