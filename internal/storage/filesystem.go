package storage

import (
	"context"
	"encoding/json"
	"errors"
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

	logger.Info("created new filesystem storage")
	return &filesystem{logger, fs}, nil
}

func (fs *filesystem) Get(_ context.Context, id string) (*Flow, error) {
	logger := fs.logger.WithField("id", id)
	logger.Debug("attempting to open file")

	file, err := fs.inner.Open(id + ".json")
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("file does not exist")
			err = nil
		}

		return nil, err
	}
	defer file.Close()

	logger.WithField("path", file.Name()).Debug("deserializing flow from file")
	f := new(Flow)
	if err := json.NewDecoder(file).Decode(f); err != nil {
		return nil, err
	}

	return f, nil
}

func (fs *filesystem) Put(_ context.Context, flow *Flow) error {
	if flow == nil {
		return errors.New("received nil flow")
	}

	logger := fs.logger.WithField("id", flow.ID)
	logger.Debug("attempting to create file")

	file, err := fs.inner.Create(flow.ID + ".json")
	if err != nil {
		return err
	}
	defer file.Close()

	logger.WithField("path", file.Name()).Debug("successfully created file")

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(flow)
}

func (fs *filesystem) Delete(_ context.Context, id string) error {
	logger := fs.logger.WithField("id", id)
	logger.Debug("attempting to remove file")

	if err := fs.inner.Remove(id + ".json"); err != nil {
		if os.IsNotExist(err) {
			logger.Debug("file already does not exist")
			err = nil
		}

		return err
	}

	return nil
}
