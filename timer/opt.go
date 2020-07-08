package timer

import (
	"time"

	"go.uber.org/zap"
)

const (
	// DefaultTimeout is the timeout in seconds set by default
	DefaultTimeout = 20 * time.Second

	// DefaultTimeoutScaling is the timeout scaling factor set by default
	DefaultTimeoutScaling = 0.5
)

// Options represent the options for a Linear Timer
type Options struct {
	Logger         *zap.Logger
	Timeout        time.Duration
	TimeoutScaling float64
}

// DefaultOptions returns the default options for a Linear Timer
func DefaultOptions() Options {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return Options{
		Logger:         logger,
		Timeout:        DefaultTimeout,
		TimeoutScaling: DefaultTimeoutScaling,
	}
}

// WithLogger updates the logger used in the Linear Timer
func (opts Options) WithLogger(logger *zap.Logger) Options {
	opts.Logger = logger
	return opts
}

// WithTimeout updates the timeout of the Linear Timer
func (opts Options) WithTimeout(timeout time.Duration) Options {
	opts.Timeout = timeout
	return opts
}

// WithTimeoutScaling updates the timeout scaling factor of the Linear Timer
func (opts Options) WithTimeoutScaling(timeoutScaling float64) Options {
	opts.TimeoutScaling = timeoutScaling
	return opts
}
