package http

import (
	"net/http"

	"github.com/akrantz01/tailfed/internal/http/requestid"
	"github.com/akrantz01/tailfed/internal/logging"
)

// Create a new HTTP server
func New() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)

	return &http.Server{
		Handler: withMiddleware(mux, requestid.Middleware, logging.Middleware),
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
