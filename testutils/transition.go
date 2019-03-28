package testutils

type InvalidTransition struct {
}

func (transition InvalidTransition) IsTransition() {}
