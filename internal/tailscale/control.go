package tailscale

import (
	"context"
	"errors"
	"net/netip"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
	"tailscale.com/client/tailscale/v2"
)

// TLSMode determines how the headscale control plane client should establish its connection
type TLSMode string

const (
	// TLSModeNone uses a plaintext connection
	TLSModeNone TLSMode = "none"
	// TLSModeInsecure uses TLS but does not verify certificates
	TLSModeInsecure TLSMode = "insecure"
	// TLSModeFull uses TLS and verifies certificates
	TLSModeFull TLSMode = "full"
)

var ErrUnknownTLSMode = errors.New("unknown TLS security mode")

// Authentication determines how the client will authenticate with the Tailscale API
type Authentication interface {
	credentials.PerRPCCredentials

	tailscale(c *tailscale.Client)
}

// ApiKey authenticates using an unrestricted API key. Where possible, prefer using OAuth credentials.
func ApiKey(key string) Authentication {
	return nil
}

// ControlPlaneOAuth authenticates using temporary credentials derived from a client ID and secret. These are limited in scope.
func ControlPlaneOAuth(clientId, clientSecret string) Authentication {
	return nil
}

// ControlPlane connects to the Tailscale control plane API
type ControlPlane interface {
	// Tailnet retrieves the name of the connected tailnet
	Tailnet() string
	// NodeInfo retrieves the details about a particular node by its ID
	NodeInfo(ctx context.Context, id string) (*NodeInfo, error)
}

// NodeInfo describes a node in the tailnet
type NodeInfo struct {
	// The ID used by the control plane
	ID string
	// The IP addresses in the network
	Addresses []netip.Addr
	// The unique public key for the machine
	Key string
	// The DNS name of the node within the network
	DNSName string
	// The hostname of the node
	Hostname string
	// The name of the Tailnet
	Tailnet string
	// The machine's operating system
	OS string
	// All ACL tags that are applied to the machine
	Tags []string
	// Whether the device is authorized to join the tailnet
	Authorized bool
	// Whether the device is shared in to the tailnet
	External bool
}

// NewControlPlane creates a new control plane client
func NewControlPlane(
	logger logrus.FieldLogger,
	backend, baseUrl, tailnet string,
	auth Authentication, tlsMode TLSMode,
) (ControlPlane, error) {
	logger = logger.WithField("backend", backend)

	switch backend {
	default:
		return nil, errors.New("unknown control plane backend")
	}
}
