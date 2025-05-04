//go:build unix

package cli

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func (r *refresh) Run(*cobra.Command, []string) error {
	pid, err := r.readPid()
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("pid file not found; check the daemon is running")
		}

		return fmt.Errorf("failed to read daemon pid file %q: %w", r.PidFile, err)
	}
	logrus.WithField("pid", pid).Debug("got daemon process id")

	if err := syscall.Kill(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send reload signal to process %d: %w", pid, err)
	}
	logrus.Info("sent reload signal to daemon")

	if !r.Wait {
		logrus.Debug("refresh wait disabled")
		return nil
	}

	logrus.Info("waiting for token to be refreshed...")

	initial, err := r.checkToken()
	if err != nil {
		return fmt.Errorf("failed to check initial token state: %w", err)
	}

	ticker := time.NewTicker(1 * time.Second)
	timer := time.NewTimer(r.Timeout)
	for {
		select {
		case <-ticker.C:
			current, err := r.checkToken()
			if err != nil {
				return fmt.Errorf("failed to check new token state: %w", err)
			}

			if slices.Equal(initial, current) {
				logrus.Debug("token not yet refreshed")
			} else {
				logrus.Info("token successfully refreshed")
				return nil
			}

		case <-timer.C:
			return fmt.Errorf("new token not issued within %s", r.Timeout)
		}
	}
}

func (r *refresh) readPid() (int, error) {
	contents, err := os.ReadFile(r.PidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.ParseInt(string(contents), 10, 64)
	if err != nil {
		return 0, err
	}

	return int(pid), nil
}

func (r *refresh) checkToken() ([]byte, error) {
	contents, err := os.ReadFile(r.Token)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	h := sha256.New()
	h.Write(contents)
	return h.Sum(nil), nil
}
