package systemd

import (
	"strings"

	"github.com/coreos/go-systemd/daemon"
	"github.com/sirupsen/logrus"
)

var notifySupported = true

// Ready notifies systemd that we are ready
func Ready() {
	notify(daemon.SdNotifyReady)
}

// Reloading notifies systemd that we are reloading
func Reloading() {
	notify(daemon.SdNotifyReloading)
}

// Stopping notifies systemd that we are stopping
func Stopping() {
	notify(daemon.SdNotifyStopping)
}

func notify(state string) {
	if !notifySupported {
		return
	}

	logger := logrus.WithField("component", "systemd")

	ok, err := daemon.SdNotify(false, state)
	if err != nil {
		logger.
			WithField("state", stateName(state)).
			WithError(err).
			Error("failed to notify systemd")
	} else if !ok {
		logger.Debug("systemd notifications are unsupported")
		notifySupported = false
	} else {
		logger.WithField("state", stateName(state)).Debug("notified systemd of new state")
	}
}

func stateName(state string) string {
	parts := strings.SplitN(state, "=", 2)
	return strings.ToLower(parts[0])
}
