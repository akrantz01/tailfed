package finalizer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/akrantz01/tailfed/internal/http/gateway"
	"github.com/akrantz01/tailfed/internal/http/lambda"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/signing"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/aws/aws-lambda-go/events"
)

// Handler responds to incoming flow finalization requests, issuing the token if the challenge was successful.
type Handler struct {
	audience string
	validity time.Duration

	signer signing.Backend
	store  storage.Backend
}

var _ gateway.Handler = (*Handler)(nil)

// New creates a new handler
func New(audience string, validity time.Duration, signer signing.Backend, store storage.Backend) *Handler {
	return &Handler{audience, validity, signer, store}
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
	} else if time.Now().After(time.Time(flow.ExpiresAt)) {
		return lambda.Error("challenge expired", http.StatusForbidden), nil
	}

	claims := signing.NewClaimsFromFlow(generateIssuer(&req.RequestContext), h.audience, h.validity, flow)
	token, err := h.signer.Sign(claims)
	if err != nil {
		logger.WithError(err).Error("failed to sign JWT")
		return lambda.Error("internal server error", http.StatusInternalServerError), nil
	}

	if err := h.store.Delete(ctx, flow.ID); err != nil {
		logger.WithError(err).Error("failed to delete flow")
		return lambda.Error("internal server error", http.StatusInternalServerError), nil
	}

	return lambda.Success(&types.FinalizeResponse{IdentityToken: token}), nil
}

func generateIssuer(ctx *events.APIGatewayProxyRequestContext) string {
	region := os.Getenv("AWS_REGION")
	expected := fmt.Sprintf("%s.execute-api.%s.amazonaws.com", ctx.APIID, region)

	if ctx.DomainName == expected {
		return "https://" + expected + "/" + ctx.Stage
	} else {
		return "https://" + ctx.DomainName
	}
}
