package consensus

import "github.com/renproject/hyperdrive/block"

type Action interface {
	IsAction()
}

type PreVote struct {
	block.PreVote
}

func (preVote PreVote) IsAction() {
}

type PreCommit struct {
	block.PreCommit
}

func (preCommit PreCommit) IsAction() {
}

type Commit struct {
	block.Commit
}

func (commit Commit) IsAction() {
}
