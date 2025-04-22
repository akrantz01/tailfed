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

func startLauncher(bus <-chan launcher.Request, store storage.Backend, tailnet string) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go launcherLoop(ctx, store, tailnet, bus)

	return cancel
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

	instance := verifier.New(&http.Client{Timeout: time.Second}, store, tailnet)

	for {
		logger.WithField("attempt", attempts).Debug("attempting verification")

		resp, err := instance.Serve(logging.WithLogger(ctx, logger), types.VerifyRequest{
			ID:      req.ID,
			Address: req.Addresses[attempts%len(req.Addresses)],
		})
		if err != nil {
			logger.WithError(err).Error("verifier execution failed")
			go markFlowFailed(ctx, store, logger, req.ID)
			return
		}

		if resp.Success {
			logger.Info("verification succeeded")
			return
		} else {
			attempts += 1
			if attempts > maxAttempts {
				logger.Error("verification failed")
				go markFlowFailed(ctx, store, logger, req.ID)
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

func markFlowFailed(ctx context.Context, store storage.Backend, logger logrus.FieldLogger, id string) {
	flow, err := store.Get(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get flow")
		return
	} else if flow == nil {
		logger.Error("flow no longer exists")
		return
	}

	flow.Status = storage.StatusFailed

	if err := store.Put(ctx, flow); err != nil {
		logger.WithError(err).Error("failed to mark flow as failed")
		return
	}

	logger.Debug("flow marked as failed")
}
