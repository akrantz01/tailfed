package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/akrantz01/tailfed/internal/finalizer"
	"github.com/akrantz01/tailfed/internal/http/gateway"
	"github.com/akrantz01/tailfed/internal/http/requestid"
	"github.com/akrantz01/tailfed/internal/initializer"
	"github.com/akrantz01/tailfed/internal/launcher"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/metadata"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/go-jose/go-jose/v4"
	"github.com/sirupsen/logrus"
)

func startGateway(tsClient tailscale.ControlPlane, launch launcher.Backend, meta metadata.Backend, signer signing.Backend, store storage.Backend) (*http.Server, <-chan error) {
	mux := http.NewServeMux()
	srv := newServer(cfg.Address, mux, requestid.Middleware, logging.Middleware)

	mux.Handle("GET /health", http.HandlerFunc(health))

	mux.Handle("GET /config.json", metadataHandler[types.Response[types.ConfigResponse]]("config.json", meta))
	mux.Handle("POST /start", lambdaHandler(initializer.New(tsClient, launch, store)))
	mux.Handle("POST /finalize", lambdaHandler(finalizer.New(cfg.Signing.Audience, cfg.Signing.Validity, signer, store)))

	mux.Handle("GET /.well-known/openid-configuration", metadataHandler[any]("openid-configuration", meta))
	mux.Handle("GET /.well-known/jwks.json", metadataHandler[jose.JSONWebKeySet]("jwks.json", meta))

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

// metadataHandler serves static keys from the metadata backend
func metadataHandler[T any](key string, meta metadata.Backend) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.FromContext(ctx)

		var data T
		if err := meta.Load(ctx, key, &data); err != nil {
			logger.WithError(err).Error("could not load from backend")
			internalServerError(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(data); err != nil {
			logger.WithError(err).Error("failed to write metadata response")
		}
	})
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
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
