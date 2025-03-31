package main

import (
	"context"
	"time"

	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/sirupsen/logrus"
)

const (
	maxAttempts   = 6
	startInterval = 500 * time.Millisecond
)

func startLauncher(store storage.Backend) (context.CancelFunc, chan<- types.VerifyRequest) {
	bus := make(chan types.VerifyRequest, 3)
	ctx, cancel := context.WithCancel(context.Background())

	go launcherLoop(ctx, store, bus)

	return cancel, bus
}

func launcherLoop(ctx context.Context, store storage.Backend, bus <-chan types.VerifyRequest) {
	logger := logrus.WithField("component", "launcher")
	logger.Debug("started local launcher")

	for {
		select {
		case req := <-bus:
			go performVerifyRequest(ctx, store, logger.WithField("flow", req.ID), req)

		case <-ctx.Done():
			logger.Debug("shut down local launcher")
			return
		}
	}
}

func performVerifyRequest(ctx context.Context, store storage.Backend, logger logrus.FieldLogger, req types.VerifyRequest) {
	logger.Info("received launch")

	attempts := 1
	wait := startInterval

	for {
		logger.WithField("attempt", attempts).Debug("attempting verification")

		// TODO: call verifier
		resp := types.VerifyResponse{
			Success: false,
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
