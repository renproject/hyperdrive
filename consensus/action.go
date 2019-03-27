package consensus

import (
	"time"

	"github.com/renproject/hyperdrive/block"
)

type Action interface {
	IsAction()
}

type Propose struct {
	block.SignedBlock
}

func (Propose) IsAction() {
}

type PreVote struct {
	block.PreVote
}

func (PreVote) IsAction() {
}

type SignedPreVote struct {
	block.SignedPreVote
}

func (SignedPreVote) IsAction() {
}

type PreCommit struct {
	block.PreCommit
}

func (PreCommit) IsAction() {
}

type SignedPreCommit struct {
	block.SignedPreCommit
}

func (SignedPreCommit) IsAction() {
}

type Commit struct {
	block.Commit
}

func (Commit) IsAction() {
}

type Timeout struct {
	time.Time
}

func (Timeout) IsAction() {
}
