package handlers

import (
	"fmt"
	"net/http"

	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/sirupsen/logrus"
)

type TokenHandler struct {
	profiles oidc.Profiles
	signer   signing.Backend
	ts       tailscale.Connection
	logger   logrus.FieldLogger
}

var _ http.Handler = (*TokenHandler)(nil)

func NewTokenHandler(profiles oidc.Profiles, signer signing.Backend, ts tailscale.Connection) *TokenHandler {
	return &TokenHandler{
		profiles: profiles,
		signer:   signer,
		ts:       ts,
		logger:   logrus.WithField("component", "http").WithField("http.handler", "token"),
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

	profileId, profile := t.profiles.GetOrDefault(r.URL.Query().Get("profile"))
	if profile == nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("unknown profile"))
		return
	}
	t.logger.WithField("profile.id", profileId).Debug("selected profile")

	claims, err := profile.GenerateClaims(issuerFor(r), node)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	token, err := t.signer.Sign(profile.AcceptedAlgorithms, claims)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		t.logger.
			WithError(err).
			WithField("profile.accepted-algorithms", profile.AcceptedAlgorithms).
			Error("failed to sign jwt")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"token":"%s"}`, token)
}
