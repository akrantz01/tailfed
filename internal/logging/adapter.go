package logging

import (
	"strings"

	"github.com/go-co-op/gocron/v2"
	"github.com/sirupsen/logrus"
)

type cronAdapter struct {
	logger *logrus.Entry
}

var _ gocron.Logger = (*cronAdapter)(nil)

// NewCronAdapter creates a new logging adapter that is compatible with gocron
func NewCronAdapter(logger *logrus.Logger) gocron.Logger {
	return &cronAdapter{
		logger: logger.WithField("logger", "gocron"),
	}
}

func (c *cronAdapter) log(msg string, args []any, emitter func(msg string, fields logrus.Fields)) {
	msg = strings.TrimPrefix(msg, "gocron: ")

	fields := make(logrus.Fields, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			fields[key] = args[i+1]
		}
	}

	emitter(msg, fields)
}

func (c *cronAdapter) Debug(msg string, args ...any) {
	c.log(msg, args, func(msg string, fields logrus.Fields) {
		c.logger.WithFields(fields).Debug(msg)
	})
}

func (c *cronAdapter) Info(msg string, args ...any) {
	c.log(msg, args, func(msg string, fields logrus.Fields) {
		c.logger.WithFields(fields).Info(msg)
	})
}

func (c *cronAdapter) Warn(msg string, args ...any) {
	c.log(msg, args, func(msg string, fields logrus.Fields) {
		c.logger.WithFields(fields).Warn(msg)
	})
}

func (c *cronAdapter) Error(msg string, args ...any) {
	c.log(msg, args, func(msg string, fields logrus.Fields) {
		c.logger.WithFields(fields).Error(msg)
	})
}
