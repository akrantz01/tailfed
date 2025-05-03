package systemd

import (
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/sirupsen/logrus"
)

// EnableWatchdog enables the systemd watchdog integration
func EnableWatchdog() {
	logger := logrus.WithField("component", "systemd")

	interval, err := daemon.SdWatchdogEnabled(false)
	if err != nil {
		logger.WithError(err).Error("failed to enable watchdog")
	} else if interval == 0 {
		logger.Debug("systemd watchdog is disabled")
	} else {
		logger.Debug("started watchdog")
		go watchdog(interval)
	}
}

func watchdog(interval time.Duration) {
	ticker := time.NewTicker(interval / 2)

	for range ticker.C {
		notify(daemon.SdNotifyWatchdog)
	}
}
