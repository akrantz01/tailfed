package main

import (
	"context"
	"net/http"
	"time"

	"github.com/akrantz01/tailfed/internal/launcher"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/akrantz01/tailfed/internal/verifier"
	"github.com/sirupsen/logrus"
)

const (
	maxAttempts   = 6
	startInterval = 500 * time.Millisecond
)

func startLauncher(store storage.Backend, tailnet string) (context.CancelFunc, chan<- launcher.Request) {
	bus := make(chan launcher.Request, 3)
	ctx, cancel := context.WithCancel(context.Background())

	go launcherLoop(ctx, store, tailnet, bus)

	return cancel, bus
}

func launcherLoop(ctx context.Context, store storage.Backend, tailnet string, bus <-chan launcher.Request) {
	logger := logrus.WithField("component", "launcher")
	logger.Debug("started local launcher")

	for {
		select {
		case req := <-bus:
			go performVerifyRequest(ctx, store, tailnet, logger.WithField("flow", req.ID), req)

		case <-ctx.Done():
			logger.Debug("shut down local launcher")
			return
		}
	}
}

func performVerifyRequest(ctx context.Context, store storage.Backend, tailnet string, logger logrus.FieldLogger, req launcher.Request) {
	logger.Info("received launch")

	attempts := 1
	wait := startInterval

	instance := verifier.New(http.DefaultClient, store, tailnet)

	for {
		logger.WithField("attempt", attempts).Debug("attempting verification")

		resp, err := instance.Serve(logging.WithLogger(ctx, logger), types.VerifyRequest{
			ID:      req.ID,
			Address: req.Addresses[attempts%len(req.Addresses)],
		})
		if err != nil {
			// TODO: disambiguate between fatal and non-fatal errors
			logger.WithError(err).Error("verifier execution failed")
			return
		}

		if resp.Success {
			logger.Info("verification succeeded")
			return
		} else {
			attempts += 1
			if attempts > maxAttempts {
				logger.Error("verification failed")
				return
			}

			logger.WithField("wait", wait).Warn("attempt failed, retrying soon")

			t := time.NewTimer(wait)
			select {
			case <-t.C:
			case <-ctx.Done():
				t.Stop()
				return
			}

			wait *= 2
		}
	}
}
