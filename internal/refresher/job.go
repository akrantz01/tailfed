package refresher

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Job performs a single run of the refresh flow
func (r *Refresher) Job(ctx context.Context) error {
	r.logger.WithField("profile", r.profile).Info("fetching token...")
	res, err := r.api.GetToken(ctx, r.profile)
	if err != nil {
		return fmt.Errorf("failed to issue token: %w", err)
	}

	r.logger.WithField("token.lifetime", res.ExpiresIn).WithField("token.expiry", res.ExpiresAt).Info("token issued")
	return r.writeToken(res.AccessToken)
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
