package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/go-jose/go-jose/v4"
	"github.com/sirupsen/logrus"
)

type JwksHandler struct {
	signer signing.Backend
	logger logrus.FieldLogger
}

var _ http.Handler = (*JwksHandler)(nil)

func NewJwksHandler(signer signing.Backend) *JwksHandler {
	return &JwksHandler{
		signer: signer,
		logger: logrus.WithField("component", "http").WithField("http.handler", "jwks"),
	}
}

func (j *JwksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, err := j.signer.PublicKey()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		j.logger.WithError(err).Error("failed to get public key")
		return
	}

	w.WriteHeader(http.StatusOK)
	body, _ := json.Marshal(jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{key},
	})
	_, _ = w.Write(body)
}
