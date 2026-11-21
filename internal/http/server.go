package http

import (
	"net/http"

	"github.com/akrantz01/tailfed/internal/http/handlers"
	"github.com/akrantz01/tailfed/internal/http/middleware"
	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/tailscale"
)

type Server = http.Server

var ErrServerClosed = http.ErrServerClosed

// NewPrivate creates an HTTP server handling every route
func NewPrivate(issuer string, profiles oidc.Profiles, signer signing.Backend, ts tailscale.Connection) *http.Server {
	return newServer(issuer, profiles, signer, ts)
}

// NewPublic creates an HTTP server handling only the routes needed to verify tokens
func NewPublic(issuer string, profiles oidc.Profiles, signer signing.Backend) *http.Server {
	return newServer(issuer, profiles, signer, nil)
}

func newServer(issuer string, profiles oidc.Profiles, signer signing.Backend, ts tailscale.Connection) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)

	if ts != nil {
		mux.Handle("GET /token", handlers.NewTokenHandler(issuer, profiles, signer, ts))
	}

	discovery := handlers.NewDiscoveryHandler(issuer, profiles)
	mux.Handle("GET /.well-known/jwks.json", handlers.NewJwksHandler(signer))
	mux.Handle("GET /.well-known/openid-configuration", discovery)
	mux.Handle("GET /{profile}/.well-known/openid-configuration", discovery)

	return &http.Server{
		Handler:     withMiddleware(mux, middleware.RequestID, middleware.FunnelSource, middleware.Logging),
		ConnContext: middleware.ConnContext,
	}
}

// NewRedirect creates an HTTP server that sends every request except health checks to target
func NewRedirect(target string) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target+r.URL.RequestURI(), http.StatusTemporaryRedirect)
	}))

	return &http.Server{
		Handler: withMiddleware(mux, middleware.RequestID, middleware.Logging),
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
