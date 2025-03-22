package logging

import (
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Initialize configures logging to standard output at the desired level
func Initialize(level string) (*logrus.Logger, error) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}

	logger := logrus.New()
	logger.SetLevel(lvl)
	logger.SetOutput(os.Stdout)
	logger.SetReportCaller(true)

	logger.SetFormatter(&logrus.TextFormatter{
		EnvironmentOverrideColors: true,

		FullTimestamp:   true,
		TimestampFormat: time.StampMilli,

		DisableLevelTruncation: true,
		PadLevelText:           true,

		QuoteEmptyFields: true,

		CallerPrettyfier: func(frame *runtime.Frame) (string, string) {
			return "", frame.File + ":" + strconv.Itoa(frame.Line)
		},
	})

	return logger, nil
}
