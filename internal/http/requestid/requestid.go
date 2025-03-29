package requestid

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey struct{}

// Get retrieves the ID of the given request
func Get(r *http.Request) string {
	return r.Context().Value(contextKey{}).(string)
}

// Middleware adds a request ID to the request
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.Must(uuid.NewV7()).String()
		ctx := context.WithValue(r.Context(), contextKey{}, id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
