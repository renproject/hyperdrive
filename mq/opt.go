package mq

import "github.com/sirupsen/logrus"

type Options struct {
	Logger      logrus.FieldLogger
	MaxCapacity int
}

func DefaultOptions() Options {
	return Options{}
}
