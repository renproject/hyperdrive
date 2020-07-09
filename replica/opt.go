package replica

import (
	"github.com/renproject/hyperdrive/mq"
	"github.com/renproject/hyperdrive/timer"

	"go.uber.org/zap"
)

// Options represent the options for a Hyperdrive Replica
type Options struct {
	Logger           *zap.Logger
	TimerOpts        timer.Options
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
		TimerOpts:        timer.DefaultOptions(),
		MessageQueueOpts: mq.DefaultOptions(),
	}
}

// WithLogger updates the logger used in the Replica with the provided logger
func (opts Options) WithLogger(logger *zap.Logger) Options {
	opts.Logger = logger
	return opts
}

// WithTimerOptions updates the Replica's timer options with the provided options
func (opts Options) WithTimerOptions(timerOpts timer.Options) Options {
	opts.TimerOpts = timerOpts
	return opts
}
