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
	"github.com/akrantz01/tailfed/internal/scheduler"
	"github.com/jonboulle/clockwork"
)

// Job performs a single run of the refresh flow
func (r *Refresher) Job(ctx context.Context) (scheduler.Schedule, error) {
	clock := clockwork.FromContext(ctx)
	issued := clock.Now()

	r.logger.WithField("profile", r.profile).Info("fetching token...")
	res, err := r.api.GetToken(ctx, r.profile)
	if err != nil {
		err = fmt.Errorf("failed to issue token: %w", err)
		if isPersistent(err) {
			return scheduler.Schedule{}, scheduler.Persistent(err)
		}
		return scheduler.Schedule{}, err
	}

	lifetime := time.Duration(res.ExpiresIn * float64(time.Second))
	schedule := scheduler.Schedule{
		Next: issued.Add(lifetime / 2),
		Max:  issued.Add(lifetime),
	}

	r.logger.
		WithField("token.lifetime", lifetime).
		WithField("token.expiry", res.ExpiresAt.Time()).
		WithField("refresh", schedule.Next).
		Info("token issued")

	if err := r.writeToken(res.AccessToken); err != nil {
		return scheduler.Schedule{}, err
	}

	return schedule, nil
}

func isPersistent(err error) bool {
	apiErr, ok := errors.AsType[*api.ApiError](err)
	if !ok {
		return false
	}

	switch apiErr.Status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return false
	default:
		return apiErr.Status >= 400 && apiErr.Status < 500
	}
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
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, strings.NewReader(token)); err != nil {
		return fmt.Errorf("failed to write token to file: %w", err)
	}

	r.logger.WithField("path", r.path).Debug("wrote token to disk")
	return nil
}
