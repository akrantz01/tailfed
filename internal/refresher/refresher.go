package refresher

import (
	"github.com/akrantz01/tailfed/internal/api"
	"github.com/sirupsen/logrus"
)

// Refresher holds the state for periodically refreshing the identity token
type Refresher struct {
	api     *api.Client
	path    string
	profile string
	logger  logrus.FieldLogger
}

// New creates a new Refresher
func New(api *api.Client, path, profile string) *Refresher {
	return &Refresher{
		api:     api,
		path:    path,
		profile: profile,
		logger:  logrus.WithField("component", "refresher"),
	}
}
