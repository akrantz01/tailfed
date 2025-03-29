package refresher

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
)

func (r *Refresher) complete(ctx context.Context, id string) {
	logger := r.logger.WithField("flow", id)

	operation := func() (string, error) {
		return r.api.Finalize(ctx, id)
	}
	notify := func(err error, next time.Duration) {
		logger.WithField("next", next).WithError(err).Warn("finalization not yet complete")
	}

	token, err := backoff.Retry(ctx, operation, backoff.WithMaxElapsedTime(1*time.Minute), backoff.WithNotify(notify))
	if err != nil {
		logger.WithError(err).Error("failed to get authorization token")
		return
	}
	logger.Debug("token successfully issued")

	r.stopServers(r.inFlight[id].servers)
	logger.Debug("shutdown callback challenge server(s)")

	if err := r.writeToken(token); err != nil {
		logger.WithField("path", r.path).WithError(err).Error("unable to write token to file")
		return
	}

	logger.Info("new token issued")
}

func (r *Refresher) writeToken(token string) error {
	baseDir := filepath.Dir(r.path)
	if err := os.MkdirAll(baseDir, os.ModeDir); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", baseDir, err)
	}

	file, err := os.Create(r.path)
	if err != nil {
		return fmt.Errorf("failed to create token file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, strings.NewReader(token)); err != nil {
		return fmt.Errorf("failed to write token to file: %w", err)
	}

	return nil
}
