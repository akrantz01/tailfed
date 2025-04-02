package verifier

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/types"
)

// Handler is triggered by a step function, performing a single verification request for an address
type Handler struct {
	client *http.Client
	store  storage.Backend

	tailnet string
}

// New creates a new handler
func New(client *http.Client, store storage.Backend, tailnet string) *Handler {
	return &Handler{client, store, tailnet}
}

func (h *Handler) Serve(ctx context.Context, req types.VerifyRequest) (*types.VerifyResponse, error) {
	logger := logging.FromContext(ctx).WithFields(map[string]any{
		"component": "verifier",
		"flow":      req.ID,
		"address":   req.Address.String(),
	})

	flow, err := h.store.Get(ctx, req.ID)
	if err != nil {
		logger.WithError(err).Error("failed to find flow")
		return &types.VerifyResponse{Success: false}, nil
	} else if flow == nil {
		logger.Error("flow no longer exists")
		return nil, fmt.Errorf("flow %q no longer exists", req.ID)
	}

	res, err := h.client.Get(fmt.Sprintf("http://%s/%s", req.Address, req.ID))
	if err != nil {
		logger.WithError(err).Error("failed to send request")
		return &types.VerifyResponse{Success: false}, nil
	}
	defer res.Body.Close()

	var challenge types.Response[types.ChallengeResponse]
	if err := json.NewDecoder(res.Body).Decode(&challenge); err != nil {
		logger.WithError(err).Error("failed to deserialize challenge response")
		return &types.VerifyResponse{Success: false}, nil
	}

	if !challenge.Success {
		logger.WithField("err", challenge.Error).Error("unsuccessful response from client")
		return &types.VerifyResponse{Success: false}, nil
	}

	expected := h.generateMac(flow)
	if !hmac.Equal(challenge.Data.Signature, expected) {
		logger.Warn("invalid signature")
		return &types.VerifyResponse{Success: false}, nil
	}

	flow.Status = storage.StatusSuccess
	if err := h.store.Put(ctx, flow); err != nil {
		logger.WithError(err).Error("failed to save flow state")
		return nil, fmt.Errorf("failed to save flow %q", flow.ID)
	}

	return &types.VerifyResponse{Success: true}, nil
}

func (h *Handler) generateMac(flow *storage.Flow) []byte {
	var buf bytes.Buffer
	buf.WriteString(h.tailnet)
	buf.WriteRune('|')
	buf.WriteString(flow.DNSName)
	buf.WriteRune('|')
	buf.WriteString(flow.PublicKey)
	buf.WriteRune('|')
	buf.WriteString(flow.OS)

	mac := hmac.New(sha256.New, flow.Secret)
	_, _ = mac.Write(buf.Bytes())
	return mac.Sum(nil)
}
