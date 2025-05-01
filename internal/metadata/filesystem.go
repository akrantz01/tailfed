package metadata

import (
	"context"
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
)

// filesystem stores data as JSON files in a directory
type filesystem struct {
	logger logrus.FieldLogger
	inner  *os.Root
}

var _ Backend = (*filesystem)(nil)

// NewFilesystem creates a new filesystem-backed storage
func NewFilesystem(logger logrus.FieldLogger, base string) (Backend, error) {
	logger = logger.WithField("base", base)

	logger.Debug("ensuring directories exist")
	if err := os.MkdirAll(base, os.ModePerm|os.ModeDir); err != nil {
		return nil, err
	}

	fs, err := os.OpenRoot(base)
	if err != nil {
		return nil, err
	}

	logger.Info("created new filesystem metadata storage")
	return &filesystem{logger, fs}, nil
}

func (fs *filesystem) Load(_ context.Context, key string, out any) error {
	logger := fs.logger.WithField("key", key)
	logger.Debug("attempting to open file")

	file, err := fs.inner.Open(key)
	if err != nil {
		return err
	}
	defer file.Close()

	logger.WithField("path", file.Name()).Debug("successfully opened file")
	return json.NewDecoder(file).Decode(out)
}

func (fs *filesystem) Save(_ context.Context, key string, data any) error {
	logger := fs.logger.WithField("key", key)
	logger.Debug("attempting to create file")

	file, err := fs.inner.Create(key)
	if err != nil {
		return err
	}
	defer file.Close()

	logger.WithField("path", file.Name()).Debug("successfully created file")

	logger.WithField("data", data).Trace("writing data to file")
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
