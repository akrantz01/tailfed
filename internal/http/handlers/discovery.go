package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/oidc"
)

type DiscoveryHandler struct {
	issuer   string
	profiles oidc.Profiles
}

var _ http.Handler = (*DiscoveryHandler)(nil)

func NewDiscoveryHandler(issuer string, profiles oidc.Profiles) *DiscoveryHandler {
	return &DiscoveryHandler{
		issuer:   issuer,
		profiles: profiles,
	}
}

func (d *DiscoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logging.FromContext(r.Context()).WithField("http.handler", "discovery")

	profileId, profile := d.profiles.GetOrDefault(r.PathValue("profile"))
	log = log.WithField("profile.id", profileId)

	if profile == nil {
		log.Debug("profile not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	log.Debug("selected profile")

	doc := oidc.NewDiscoveryDocument(d.issuer, profile.AcceptedAlgorithms)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doc)
}
