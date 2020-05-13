package timer

import (
	"io"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DefaultTimeout        = 20 * time.Second
	DefaultTimeoutScaling = 0.5
)

type Options struct {
	Logger         logrus.FieldLogger
	Timeout        time.Duration
	TimeoutScaling float64
}

func DefaultOptions() Options {
	return Options{
		Logger:         loggerWithFields(logrus.New()),
		Timeout:        DefaultTimeout,
		TimeoutScaling: DefaultTimeoutScaling,
	}
}

func (opts Options) WithLogLevel(level logrus.Level) Options {
	logger := logrus.New()
	logger.SetLevel(level)
	opts.Logger = loggerWithFields(logger)
	return opts
}

func (opts Options) WithLogOutput(output io.Writer) Options {
	logger := logrus.New()
	logger.SetOutput(output)
	opts.Logger = loggerWithFields(logger)
	return opts
}

func (opts Options) WithTimeout(timeout time.Duration) Options {
	opts.Timeout = timeout
	return opts
}

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
