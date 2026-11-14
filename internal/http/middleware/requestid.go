package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey struct{}

// GetRequestID retrieves the ID of the given request
func GetRequestID(r *http.Request) string {
	return r.Context().Value(contextKey{}).(string)
}

// RequestID adds a unique ID to the request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.Must(uuid.NewV7()).String()
		ctx := context.WithValue(r.Context(), contextKey{}, id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
