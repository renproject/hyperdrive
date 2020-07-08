package timer

import (
	"io"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// DefaultTimeout is the timeout in seconds set by default
	DefaultTimeout = 20 * time.Second

	// DefaultTimeoutScaling is the timeout scaling factor set by default
	DefaultTimeoutScaling = 0.5
)

// Options represent the options for a Linear Timer
type Options struct {
	Logger         logrus.FieldLogger
	Timeout        time.Duration
	TimeoutScaling float64
}

// DefaultOptions returns the default options for a Linear Timer
func DefaultOptions() Options {
	return Options{
		Logger:         loggerWithFields(logrus.New()),
		Timeout:        DefaultTimeout,
		TimeoutScaling: DefaultTimeoutScaling,
	}
}

// WithLogLevel updates the log level of the Linear Timer's logger
func (opts Options) WithLogLevel(level logrus.Level) Options {
	logger := logrus.New()
	logger.SetLevel(level)
	opts.Logger = loggerWithFields(logger)
	return opts
}

// WithLogOutput updates where the Linear Timer's logger will log data to
func (opts Options) WithLogOutput(output io.Writer) Options {
	logger := logrus.New()
	logger.SetOutput(output)
	opts.Logger = loggerWithFields(logger)
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

func loggerWithFields(logger *logrus.Logger) logrus.FieldLogger {
	return logger.
		WithField("lib", "hyperdrive").
		WithField("pkg", "timer").
		WithField("com", "timer")
}
