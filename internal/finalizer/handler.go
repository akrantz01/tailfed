package finalizer

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/akrantz01/tailfed/internal/http/gateway"
	"github.com/akrantz01/tailfed/internal/http/lambda"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/aws/aws-lambda-go/events"
)

// Handler responds to incoming flow finalization requests, issuing the token if the challenge was successful.
type Handler struct {
	store storage.Backend
}

var _ gateway.Handler = (*Handler)(nil)

// New creates a new handler
func New(store storage.Backend) *Handler {
	return &Handler{store}
}

func (h *Handler) Serve(ctx context.Context, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	logger := logging.FromContext(ctx).WithField("component", "logger")

	var body types.FinalizeRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return lambda.Error("invalid request body", http.StatusUnprocessableEntity), nil
	}
	logger.WithField("body", body).Debug("")

	flow, err := h.store.Get(ctx, body.ID)
	if err != nil {
		logger.WithError(err).Error("failed to get flow")
		return lambda.Error("internal server error", http.StatusInternalServerError), nil
	} else if flow == nil {
		logger.Warn("flow not found")
		return lambda.Error("flow not found", http.StatusNotFound), nil
	}

	if flow.Status != storage.StatusSuccess {
		return lambda.Error("challenge not verified", http.StatusForbidden), nil
	}

	// TODO: generate and sign jwt
	token := "this.will.be.the.token"

	if err := h.store.Delete(ctx, flow.ID); err != nil {
		logger.WithError(err).Error("failed to delete flow")
		return lambda.Error("internal server error", http.StatusInternalServerError), nil
	}

	return lambda.Success(&types.FinalizeResponse{IdentityToken: token}), nil
}
