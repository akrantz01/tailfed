package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/akrantz01/tailfed/internal/http/gateway"
	"github.com/akrantz01/tailfed/internal/http/requestid"
	"github.com/akrantz01/tailfed/internal/initializer"
	"github.com/akrantz01/tailfed/internal/launcher"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/sirupsen/logrus"
)

func startGateway(tsClient *tailscale.API, launch launcher.Backend, store storage.Backend) (*http.Server, <-chan error) {
	mux := http.NewServeMux()
	srv := newServer(cfg.Address, mux, requestid.Middleware, logging.Middleware)

	mux.Handle("POST /start", lambdaHandler(initializer.New(tsClient, launch, store)))
	// TODO: register finalize handler

	serverErrors := make(chan error, 1)
	go func() {
		logrus.WithField("address", cfg.Address).Info("server is ready")
		serverErrors <- srv.ListenAndServe()
	}()

	return srv, serverErrors
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

	resp := types.Response[struct{}]{
		Success: false,
		Error:   "internal server error",
	}
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		logrus.WithError(err).Error("failed to write internal server error response")
	}
}
