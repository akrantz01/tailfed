package handlers

import (
	"net/http"

	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/sirupsen/logrus"
)

type TokenHandler struct {
	signer signing.Backend
	ts     tailscale.Connection
	logger logrus.FieldLogger
}

var _ http.Handler = (*TokenHandler)(nil)

func NewTokenHandler(signer signing.Backend, ts tailscale.Connection) *TokenHandler {
	return &TokenHandler{
		signer: signer,
		ts:     ts,
		logger: logrus.WithField("component", "http").WithField("http.handler", "token"),
	}
}

func (t *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	node, err := t.ts.WhoIs(r.Context(), r.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		t.logger.
			WithError(err).
			WithField("address", r.RemoteAddr).
			Error("failed to query tailscale for remote address")
		return
	} else if node == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	t.logger.
		WithField("node", node).
		Infof("found node for %s", r.RemoteAddr)

	w.WriteHeader(http.StatusOK)
}
