package generator

import (
	"context"
	"sync"
	"time"

	"github.com/akrantz01/tailfed/internal/metadata"
	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/go-jose/go-jose/v4"
)

// Handler is triggered by EventBridge once a day, generating the OIDC metadata
type Handler struct {
	validity time.Duration

	meta   metadata.Backend
	signer signing.Backend
}

// New creates a new handler
func New(validity time.Duration, meta metadata.Backend, signer signing.Backend) *Handler {
	return &Handler{validity, meta, signer}
}

func (h *Handler) Serve(ctx context.Context, req types.GenerateRequest) error {
	var wg sync.WaitGroup

	configErrCh := wrapWriteJob(ctx, req, &wg, h.writeConfig)

	jwkErrCh := wrapWriteJob(ctx, req, &wg, h.writeJwkSet)
	discoveryDocumentErrCh := wrapWriteJob(ctx, req, &wg, h.writeDiscoveryDocument)

	wg.Wait()
	return combineErrors(configErrCh, jwkErrCh, discoveryDocumentErrCh)
}

func (h *Handler) writeConfig(ctx context.Context, _ types.GenerateRequest) error {
	return h.meta.Save(ctx, "config.json", &types.Response[types.ConfigResponse]{
		Success: true,
		Data: &types.ConfigResponse{
			Frequency: (h.validity / 4) * 3, // Refresh after 75% of the duration has elapsed
		},
	})
}

func (h *Handler) writeJwkSet(ctx context.Context, _ types.GenerateRequest) error {
	key, err := h.signer.PublicKey()
	if err != nil {
		return err
	}

	return h.meta.Save(ctx, "jwks.json", jose.JSONWebKeySet{Keys: []jose.JSONWebKey{key}})
}

func (h *Handler) writeDiscoveryDocument(ctx context.Context, req types.GenerateRequest) error {
	doc := oidc.NewDiscoveryDocument(req.Issuer)
	return h.meta.Save(ctx, "openid-configuration", doc)
}
