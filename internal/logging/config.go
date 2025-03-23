package logging

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Initialize configures logging to standard output at the desired level
func Initialize(levelName string) error {
	logrus.SetOutput(os.Stdout)
	logrus.SetReportCaller(true)

	logrus.SetFormatter(&logrus.TextFormatter{
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

	level, err := logrus.ParseLevel(levelName)
	if err != nil {
		return fmt.Errorf("invalid log level %q", levelName)
	}
	logrus.SetLevel(level)

	return nil
}
