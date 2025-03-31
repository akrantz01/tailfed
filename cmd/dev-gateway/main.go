package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/akrantz01/tailfed/internal/configloader"
	"github.com/akrantz01/tailfed/internal/http/gateway"
	"github.com/akrantz01/tailfed/internal/http/requestid"
	"github.com/akrantz01/tailfed/internal/initializer"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:           "dev-gateway",
		Short:         "An API gateway implementation for local development",
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       preRun,
		RunE:          run,
	}
	cmd.Flags().StringP("log-level", "l", "info", "The minimum level to log at (choices: panic, fatal, error, warn, info, debug, trace)")
	cmd.Flags().StringP("address", "a", "127.0.0.1:8000", "The address and port combination to listen on")

	cmd.Flags().StringP("storage.backend", "b", "filesystem", "Where to store data for in-flight flows (choices: filesystem)")
	cmd.Flags().String("storage.path", "flows", "The directory path used by the filesystem backend")

	cmd.Flags().String("tailscale.tailnet", "", "The name of the tailnet to issue tokens for")
	cmd.Flags().String("tailscale.api-key", "", "The Tailscale API key to authenticate with")
	cmd.Flags().String("tailscale.oauth.client-id", "", "The Tailscale OAuth client ID to authenticate with")
	cmd.Flags().String("tailscale.oauth.client-secret", "", "The Tailscale OAuth client secret to authenticate with")

	err := cmd.Execute()
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

// preRun configures the runtime environment
func preRun(cmd *cobra.Command, _ []string) error {
	err := configloader.LoadInto(&cfg,
		configloader.WithFlags(cmd.Flags()),
		configloader.WithEnvPrefix("DEV_GATEWAY_"),
	)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := logging.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return nil
}

// run configures and launches the development gateway
func run(*cobra.Command, []string) error {
	store, err := cfg.Storage.NewBackend()
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}

	tsClient := cfg.Tailscale.NewClient()

	mux := http.NewServeMux()
	srv := newServer(cfg.Address, mux, requestid.Middleware, logging.Middleware)

	mux.Handle("POST /start", lambdaHandler(initializer.New(tsClient, store)))
	// TODO: register finalize handler

	serverErrors := make(chan error, 1)
	go func() {
		logrus.WithField("address", cfg.Address).Info("server is ready")
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Fatal("failed to start server")
		}

	case sig := <-shutdown:
		logrus.WithField("signal", sig.String()).Info("signal received, shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logrus.WithError(err).Fatal("failed to shutdown server")
		}
	}

	logrus.Info("successfully shutdown, goodbye!")
	return nil
}

// newServer creates a new HTTP server with a base handler and a sequence of middleware. Middleware are applied such
// that the first passed is the first to execute and last to return.
func newServer(address string, handler http.Handler, middleware ...func(http.Handler) http.Handler) *http.Server {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	return &http.Server{
		Addr:    address,
		Handler: handler,
	}
}

// lambdaHandler acts as an adapter between the AWS lambda handler functions and HTTP handler functions
func lambdaHandler(next gateway.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logging.FromContext(r.Context())

		req, err := gateway.FromHttpRequest(r)
		if err != nil {
			logger.WithError(err).Error("failed to convert to gateway request")
			internalServerError(w)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response, err := next.Serve(ctx, req)
		if err != nil {
			logger.WithError(err).Error("lambda handler failed")
			internalServerError(w)
			return
		}

		if err := gateway.WriteHttpResponse(w, response); err != nil {
			logger.WithError(err).Error("failed to write lambda response")
		}
	})
}

func internalServerError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	resp := api.Response[struct{}]{
		Success: false,
		Error:   "internal server error",
	}
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		logrus.WithError(err).Error("failed to write internal server error response")
	}
}
