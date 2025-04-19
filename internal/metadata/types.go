package metadata

import (
	"context"
)

// Backend provides a mechanism for storing OpenID Connect metadata
type Backend interface {
	// Load reads and deserializes metadata from JSON. Only used for local development
	Load(ctx context.Context, key string, out any) error
	// Save stores a new set of metadata. The metadata must be serializable to JSON
	Save(ctx context.Context, key string, data any) error
}
