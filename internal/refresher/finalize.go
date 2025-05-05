package refresher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/cenkalti/backoff/v5"
)

func (r *Refresher) complete(ctx context.Context, id string) {
	logger := r.logger.WithField("flow", id)
	defer r.stopServersFor(id)

	operation := func() (string, error) {
		token, err := r.api.Finalize(ctx, id)
		if err == nil {
			return token, nil
		}

		var httpErr *api.Error
		if errors.As(err, &httpErr) && httpErr.StatusCode() != http.StatusConflict {
			err = backoff.Permanent(err)
		}
		return "", err
	}
	notify := func(err error, next time.Duration) {
		logger.WithField("next", next).WithError(err).Warn("finalization not yet complete")
	}

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second

	token, err := backoff.Retry(ctx, operation,
		backoff.WithBackOff(expBackoff),
		backoff.WithMaxElapsedTime(3*time.Minute),
		backoff.WithNotify(notify),
	)
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
	if err := os.MkdirAll(baseDir, 0o755|os.ModeDir); err != nil {
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

func (r *Refresher) stopServersFor(id string) {
	inFlight, ok := r.inFlight[id]
	if !ok {
		return
	}

	r.stopServers(inFlight.servers)
}
