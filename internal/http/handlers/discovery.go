package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/sirupsen/logrus"
)

type DiscoveryHandler struct {
	profiles oidc.Profiles
	logger   logrus.FieldLogger
}

var _ http.Handler = (*DiscoveryHandler)(nil)

func NewDiscoveryHandler(profiles oidc.Profiles) *DiscoveryHandler {
	return &DiscoveryHandler{
		profiles: profiles,
		logger:   logrus.WithField("component", "http").WithField("http.handler", "discovery"),
	}
}

func (d *DiscoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	profileId, profile := d.profiles.GetOrDefault(r.PathValue("profile"))
	if profile == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	d.logger.WithField("profile.id", profileId).Debug("selected profile")

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
