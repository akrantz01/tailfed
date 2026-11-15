package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/akrantz01/tailfed/internal/http/types"
	"github.com/akrantz01/tailfed/internal/logging"
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
	log := logging.FromContext(r.Context()).WithField("http.handler", "token")

	if r.Header.Get("Sec-Tailfed") != "1" {
		log.Info("missing Sec-Tailfed header")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("header 'Sec-Tailfed' must be set to '1'"))
		return
	}

	node, err := t.ts.WhoIs(r.Context(), r.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.
			WithError(err).
			Error("failed to query tailscale for remote address")
		return
	} else if node == nil {
		log.Info("address not on tailnet")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	profileId, profile := t.profiles.GetOrDefault(r.URL.Query().Get("profile"))
	log = log.WithField("profile.id", profileId)

	if profile == nil {
		log.Warn("profile does not exist")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("unknown profile"))
		return
	}
	log.Debug("selected profile")

	claims, err := profile.GenerateClaims(issuerFor(r), node)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.WithError(err).Error("failed to generate claims")
		return
	}

	token, err := t.signer.Sign(profile.AcceptedAlgorithms, claims)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.
			WithError(err).
			WithField("profile.accepted-algorithms", profile.AcceptedAlgorithms).
			Error("failed to sign jwt")
		return
	}

	log.
		WithField("token.validity", profile.Validity).
		WithField("token.expires", claims.Expiry.Time()).
		Info("issued new token")

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(types.TokenResponse{
		TokenType:   "Bearer",
		AccessToken: token,
		ExpiresIn:   profile.Validity.Seconds(),
		ExpiresAt:   *claims.Expiry,
	})
}
