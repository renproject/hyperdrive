package replica

import (
	"github.com/renproject/hyperdrive/mq"
	"github.com/renproject/hyperdrive/process"

	"go.uber.org/zap"
)

// Options represent the options for a Hyperdrive Replica
type Options struct {
	Logger           *zap.Logger
	StartingHeight   process.Height
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
		StartingHeight:   process.DefaultHeight,
		MessageQueueOpts: mq.DefaultOptions(),
	}
}

// WithLogger updates the logger used in the Replica with the provided logger
func (opts Options) WithLogger(logger *zap.Logger) Options {
	opts.Logger = logger
	return opts
}

// WithStartingHeight updates the height that the Replica will start at
func (opts Options) WithStartingHeight(height process.Height) Options {
	opts.StartingHeight = height
	return opts
}

// WithMqOptions updates the Replica's message queue options
func (opts Options) WithMqOptions(mqOpts mq.Options) Options {
	opts.MessageQueueOpts = mqOpts
	return opts
}
