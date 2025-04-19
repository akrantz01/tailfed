package generator

import (
	"context"
	"sync"

	"github.com/akrantz01/tailfed/internal/metadata"
	"github.com/akrantz01/tailfed/internal/oidc"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/go-jose/go-jose/v4"
)

// Handler is triggered by EventBridge once a day, generating the OIDC metadata
type Handler struct {
	meta   metadata.Backend
	signer signing.Backend
}

// New creates a new handler
func New(meta metadata.Backend, signer signing.Backend) *Handler {
	return &Handler{meta, signer}
}

func (h *Handler) Serve(ctx context.Context, req types.GenerateRequest) error {
	var wg sync.WaitGroup

	jwkErrCh := wrapWriteJob(ctx, req, &wg, h.writeJwkSet)
	discoveryDocumentErrCh := wrapWriteJob(ctx, req, &wg, h.writeDiscoveryDocument)

	wg.Wait()
	return combineErrors(jwkErrCh, discoveryDocumentErrCh)
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
