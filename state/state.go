package state

type State interface {
	IsState()
}

type WaitingForPropose struct {
}

func (WaitingForPropose) IsState() {}

type WaitingForPolka struct {
}

func (WaitingForPolka) IsState() {}

type WaitingForCommit struct {
}

func (WaitingForCommit) IsState() {}
