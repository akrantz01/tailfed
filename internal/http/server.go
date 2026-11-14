package http

import (
	"net/http"

	"github.com/akrantz01/tailfed/internal/http/handlers"
	"github.com/akrantz01/tailfed/internal/http/middleware"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/tailscale"
)

// Create a new HTTP server
func New(signer signing.Backend, ts tailscale.Connection) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)

	mux.Handle("GET /token", handlers.NewTokenHandler(signer, ts))

	mux.Handle("GET /.well-known/openid-configuration", handlers.NewDiscoveryHandler())

	return &http.Server{
		Handler: withMiddleware(mux, middleware.RequestID, logging.Middleware),
	}
}

func withMiddleware(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	return handler
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
