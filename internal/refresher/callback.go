package refresher

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/akrantz01/tailfed/internal/types"
	"github.com/sirupsen/logrus"
)

type challengeHandler struct {
	logger    logrus.FieldLogger
	refresher *Refresher

	id     string
	path   string
	secret []byte
}

func (r *Refresher) launchServer(id string, secret []byte, l net.Listener) *http.Server {
	logger := logrus.WithFields(map[string]any{
		"component": "refresher.server",
		"address":   l.Addr().String(),
		"flow":      id,
	})
	s := &http.Server{
		Handler: &challengeHandler{
			logger:    logger,
			refresher: r,

			id:     id,
			path:   "/" + id,
			secret: secret,
		},
	}

	go func() {
		logger.Debug("started challenge server")

		err := s.Serve(l)
		if errors.Is(err, http.ErrServerClosed) {
			logger.Debug("server shutdown")
		} else if err != nil {
			logger.WithError(err).Error("server failed")
		}
	}()

	return s
}

func (ch *challengeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != ch.path {
		apiError(w, "not found", http.StatusNotFound)
		return
	} else if r.Method != http.MethodGet {
		apiError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := ch.refresher.ts.Status(r.Context())
	if err != nil {
		ch.logger.WithError(err).Error("failed to get node status")
		apiError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	buf.WriteString(status.Tailnet)
	buf.WriteRune('|')
	buf.WriteString(status.DNSName)
	buf.WriteRune('|')
	buf.WriteString(status.PublicKey)
	buf.WriteRune('|')
	buf.WriteString(status.OS)

	mac := hmac.New(sha256.New, ch.secret)
	_, _ = mac.Write(buf.Bytes())
	signature := mac.Sum(nil)

	response(w, &types.Response[types.ChallengeResponse]{
		Success: true,
		Data:    &types.ChallengeResponse{Signature: signature},
	}, 200)
}

func apiError(w http.ResponseWriter, message string, status int) {
	response(w, &types.Response[struct{}]{
		Success: false,
		Error:   message,
	}, status)
}

func response[R any](w http.ResponseWriter, res *types.Response[R], status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		logrus.
			WithFields(map[string]any{
				"component": "refresher.server",
				"response":  res,
				"status":    status,
			}).
			WithError(err).
			Error("failed to write response")
	}
}
