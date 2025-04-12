package metadata

import (
	"context"
	"encoding/json"
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

func (fs *filesystem) Load(_ context.Context, key string, out any) error {
	file, err := fs.inner.Open(key)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(out)
}

func (fs *filesystem) Save(_ context.Context, key string, data any) error {
	file, err := fs.inner.Create(key)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
