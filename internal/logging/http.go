package logging

import (
	"context"
	"net/http"
	"time"

	"github.com/akrantz01/tailfed/internal/http/requestid"
	"github.com/sirupsen/logrus"
)

// Middleware adds a layer for logging basic request information
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{w, http.StatusOK}

		logger := logrus.WithFields(map[string]any{
			"id":     requestid.Get(r),
			"method": r.Method,
			"path":   r.URL.Path,
		})
		ctx := context.WithValue(r.Context(), contextKey{}, logger)

		start := time.Now()

		logger.Info("new request")
		next.ServeHTTP(recorder, r.WithContext(ctx))

		logger.
			WithFields(map[string]any{
				"status":   recorder.status,
				"duration": time.Since(start),
			}).
			Info("request finished")
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

var _ http.ResponseWriter = (*statusRecorder)(nil)

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
