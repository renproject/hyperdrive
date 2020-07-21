package replica

import (
	"github.com/renproject/hyperdrive/mq"

	"go.uber.org/zap"
)

// Options represent the options for a Hyperdrive Replica
type Options struct {
	Logger           *zap.Logger
	MessageQueueOpts mq.Options
}

// DefaultOptions returns the default options for a Hyperdrive Replica
func DefaultOptions() Options {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return Options{
		Logger:           logger,
		MessageQueueOpts: mq.DefaultOptions(),
	}
}

// WithLogger updates the logger used in the Replica with the provided logger
func (opts Options) WithLogger(logger *zap.Logger) Options {
	opts.Logger = logger
	return opts
}

// WithMqOptions updates the Replica's message queue options
func (opts Options) WithMqOptions(mqOpts mq.Options) Options {
	opts.MessageQueueOpts = mqOpts
	return opts
}
