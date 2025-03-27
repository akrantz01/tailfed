package refresher

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/sirupsen/logrus"
)

type challengeHandler struct {
	ts *tailscale.Client

	path   string
	secret string
}

func (r *Refresher) launchServer(id, secret string, l net.Listener) *http.Server {
	s := &http.Server{
		Handler: &challengeHandler{
			ts:     r.ts,
			path:   "/" + id,
			secret: secret,
		},
	}

	go func() {
		logger := r.logger.WithFields(map[string]any{
			"address":   l.Addr().String(),
			"component": "refresher.server",
		})
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

	status, err := ch.ts.Status(r.Context())
	if err != nil {
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

	mac := hmac.New(sha256.New, []byte(ch.secret))
	_, _ = mac.Write(buf.Bytes())
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	response(w, &api.Response[api.ChallengeResponse]{
		Success: true,
		Data:    &api.ChallengeResponse{Signature: signature},
	}, 200)
}

func apiError(w http.ResponseWriter, message string, status int) {
	response(w, &api.Response[struct{}]{
		Success: false,
		Error:   message,
	}, status)
}

func response[R any](w http.ResponseWriter, res *api.Response[R], status int) {
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
