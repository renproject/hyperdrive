package replica

import (
	"io"

	"github.com/renproject/hyperdrive/mq"
	"github.com/renproject/hyperdrive/timer"
	"github.com/sirupsen/logrus"
)

type Options struct {
	Logger           logrus.FieldLogger
	TimerOpts        timer.Options
	MessageQueueOpts mq.Options
}

func DefaultOptions() Options {
	return Options{
		Logger:           loggerWithFields(logrus.New()),
		TimerOpts:        timer.DefaultOptions(),
		MessageQueueOpts: mq.DefaultOptions(),
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

func (opts Options) WithTimerOptions(timerOpts timer.Options) Options {
	opts.TimerOpts = timerOpts
	return opts
}

func loggerWithFields(logger *logrus.Logger) logrus.FieldLogger {
	return logger.
		WithField("lib", "hyperdrive").
		WithField("pkg", "replica").
		WithField("com", "replica")
}
