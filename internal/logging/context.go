package logging

import (
	"context"

	"github.com/sirupsen/logrus"
)

type contextKey struct{}

// FromContext retrieves a logger from the context, or the global logger if one is not present
func FromContext(ctx context.Context) logrus.FieldLogger {
	l, ok := ctx.Value(contextKey{}).(logrus.FieldLogger)
	if !ok {
		return logrus.StandardLogger()
	}

	return l
}
