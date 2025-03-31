package initializer

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"time"

	"github.com/akrantz01/tailfed/internal/api"
	"github.com/akrantz01/tailfed/internal/http/gateway"
	"github.com/akrantz01/tailfed/internal/http/lambda"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

// Handler responds to incoming flow start requests, performing the necessary validations before issuing a challenge and
// kicking off the verification workflow.
type Handler struct {
	store storage.Backend
	ts    *tailscale.API
}

var _ gateway.Handler = (*Handler)(nil)

// New creates a new handler
func New(client *tailscale.API, store storage.Backend) *Handler {
	return &Handler{
		store: store,
		ts:    client,
	}
}

func (h *Handler) Serve(ctx context.Context, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	logger := logging.FromContext(ctx)

	var body api.StartRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return lambda.Error("invalid request body", http.StatusUnprocessableEntity), nil
	}
	logger.WithField("body", body).Debug("")

	if len(body.PortBindings) != 2 {
		return lambda.Error("must have two port bindings", http.StatusUnprocessableEntity), nil
	}

	info, err := h.ts.NodeInfo(ctx, body.Node)
	if err != nil {
		logger.WithError(err).Error("getting node info failed")
		return lambda.Error("internal server error", http.StatusInternalServerError), nil
	} else if info == nil {
		logger.Warn("attempt to start token issuance for non-existent node")
		return lambda.Error("node not found", http.StatusUnprocessableEntity), nil
	}

	id := uuid.Must(uuid.NewV7()).String()

	secret := make([]byte, 64)
	if _, err := rand.Read(secret); err != nil {
		logger.WithError(err).Error("failed to generate signing secret")
		return lambda.Error("internal server error", http.StatusInternalServerError), nil
	}

	err = h.store.Put(ctx, &storage.Flow{
		ID:        id,
		Status:    storage.StatusPending,
		ExpiresAt: storage.UnixTime(time.Now().UTC().Add(5 * time.Minute)),
		Secret:    secret,
		Node:      info.ID,
		PublicKey: info.Key,
		DNSName:   info.DNSName,
		OS:        info.OS,
	})
	if err != nil {
		logger.WithError(err).Error("failed to save flow")
		return lambda.Error("internal server error", http.StatusInternalServerError), nil
	}

	// TODO: launch challenge verifier(s)

	return lambda.Success(&api.StartResponse{ID: id, SigningSecret: secret}), nil
}
