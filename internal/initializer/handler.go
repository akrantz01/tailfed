package initializer

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/akrantz01/tailfed/internal/http/gateway"
	"github.com/akrantz01/tailfed/internal/http/lambda"
	"github.com/akrantz01/tailfed/internal/launcher"
	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/akrantz01/tailfed/internal/tailscale"
	"github.com/akrantz01/tailfed/internal/types"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

// Handler responds to incoming flow start requests, performing the necessary validations before issuing a challenge and
// kicking off the verification workflow.
type Handler struct {
	launch launcher.Backend
	store  storage.Backend
	ts     tailscale.ControlPlane
}

var _ gateway.Handler = (*Handler)(nil)

// New creates a new handler
func New(client tailscale.ControlPlane, launch launcher.Backend, store storage.Backend) *Handler {
	return &Handler{
		store:  store,
		launch: launch,
		ts:     client,
	}
}

func (h *Handler) Serve(ctx context.Context, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	logger := logging.FromContext(ctx).WithField("component", "logger")

	var body types.StartRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return lambda.Error("invalid request body", http.StatusUnprocessableEntity), nil
	}
	logger.WithField("body", body).Debug("")

	if body.Ports.IPv4 == 0 || body.Ports.IPv6 == 0 {
		return lambda.Error("must have two port bindings", http.StatusUnprocessableEntity), nil
	}

	info, err := h.ts.NodeInfo(ctx, body.Node)
	if err != nil {
		logger.WithError(err).Error("getting node info failed")
		return lambda.InternalServerError(), nil
	} else if info == nil {
		logger.Warn("attempt to start token issuance for non-existent node")
		return lambda.Error("node not found", http.StatusUnprocessableEntity), nil
	}

	if len(info.Addresses) != 2 {
		logger.Errorf("expected 2 addresses, got %d", len(info.Addresses))
		return lambda.InternalServerError(), nil
	}

	id := uuid.Must(uuid.NewV7()).String()

	secret := make([]byte, 64)
	if _, err := rand.Read(secret); err != nil {
		logger.WithError(err).Error("failed to generate signing secret")
		return lambda.InternalServerError(), nil
	}

	dnsNameParts := strings.Split(info.DNSName, ".")

	err = h.store.Put(ctx, &storage.Flow{
		ID:          id,
		Status:      storage.StatusPending,
		ExpiresAt:   storage.UnixTime(time.Now().UTC().Add(5 * time.Minute)),
		Secret:      secret,
		Node:        info.ID,
		PublicKey:   info.Key,
		DNSName:     info.DNSName,
		MachineName: dnsNameParts[0],
		Hostname:    info.Hostname,
		Tailnet:     info.Tailnet,
		OS:          info.OS,
		Tags:        info.Tags,
		Authorized:  info.Authorized,
		External:    info.External,
	})
	if err != nil {
		logger.WithError(err).Error("failed to save flow")
		return lambda.InternalServerError(), nil
	}

	addresses := make([]netip.AddrPort, 0, 2)
	for _, address := range info.Addresses {
		var port uint16
		if address.Is4() {
			port = body.Ports.IPv4
		} else {
			port = body.Ports.IPv6
		}

		addresses = append(addresses, netip.AddrPortFrom(address, port))
	}

	if err := h.launch.Launch(id, addresses); err != nil {
		logger.WithError(err).Error("failed to launch verifier")
		return lambda.InternalServerError(), nil
	}

	return lambda.Success(&types.StartResponse{ID: id, SigningSecret: secret}), nil
}
