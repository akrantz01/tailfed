package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
)

// filesystem stores data as JSON files in a directory
type filesystem struct {
	inner *os.Root
}

var _ Backend = (*filesystem)(nil)

// NewFilesystem creates a new filesystem-backed storage
func NewFilesystem(base string) (Backend, error) {
	if err := os.MkdirAll(base, os.ModePerm|os.ModeDir); err != nil {
		return nil, err
	}

	fs, err := os.OpenRoot(base)
	if err != nil {
		return nil, err
	}

	return &filesystem{fs}, nil
}

func (fs *filesystem) Get(_ context.Context, id string) (*Flow, error) {
	file, err := fs.inner.Open(id + ".json")
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}

		return nil, err
	}
	defer file.Close()

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

	file, err := fs.inner.Create(flow.ID + ".json")
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(flow)
}

func (fs *filesystem) Delete(_ context.Context, id string) error {
	if err := fs.inner.Remove(id + ".json"); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}

		return err
	}

	return nil
}
