package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/oidc"
)

type DiscoveryHandler struct {
	profiles oidc.Profiles
}

var _ http.Handler = (*DiscoveryHandler)(nil)

func NewDiscoveryHandler(profiles oidc.Profiles) *DiscoveryHandler {
	return &DiscoveryHandler{
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

	doc := oidc.NewDiscoveryDocument(issuerFor(r), profile.AcceptedAlgorithms)

	w.WriteHeader(http.StatusOK)
	body, _ := json.Marshal(doc)
	_, _ = w.Write(body)
}

func issuerFor(r *http.Request) string {
	var scheme string
	if r.TLS == nil {
		scheme = "http"
	} else {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
