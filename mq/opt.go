package mq

import "go.uber.org/zap"

// Options define the Message Queue options
type Options struct {
	Logger      *zap.Logger
	MaxCapacity int
}

// DefaultOptions returns the default options as used by the Message Queue
func DefaultOptions() Options {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return Options{
		Logger:      logger,
		MaxCapacity: 1000,
	}
}

// WithLogger updates the logger used in the Message Queue
func (opts Options) WithLogger(logger *zap.Logger) Options {
	opts.Logger = logger
	return opts
}

// WithMaxCapacity updates the maximum capacity of the Message Queue
func (opts Options) WithMaxCapacity(capacity int) Options {
	opts.MaxCapacity = capacity
	return opts
}
