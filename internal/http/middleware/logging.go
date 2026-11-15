package middleware

import (
	"net/http"
	"time"

	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/sirupsen/logrus"
)

// Logging adds a layer for logging basic request information
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{w, http.StatusOK}

		logger := logrus.
			WithField("component", "http").
			WithField("http.request.id", GetRequestID(r)).
			WithField("http.request.method", r.Method).
			WithField("http.request.path", r.URL.Path).
			WithField("http.request.remote-addr", r.RemoteAddr)
		ctx := logging.WithLogger(r.Context(), logger)

		start := time.Now()

		logger.Info("new request")
		next.ServeHTTP(recorder, r.WithContext(ctx))

		logger.
			WithField("http.response.status", recorder.status).
			WithField("http.response.duration", time.Since(start)).
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
