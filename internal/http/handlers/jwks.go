package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/go-jose/go-jose/v4"
)

type JwksHandler struct {
	signer signing.Backend
}

var _ http.Handler = (*JwksHandler)(nil)

func NewJwksHandler(signer signing.Backend) *JwksHandler {
	return &JwksHandler{
		signer: signer,
	}
}

func (j *JwksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{
		Keys: j.signer.PublicKeys(),
	})
}
