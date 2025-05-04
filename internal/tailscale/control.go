package tailscale

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"

	headscale "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"tailscale.com/client/tailscale/v2"
)

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

// HostedControlPlane connects to the Tailscale control plane
type HostedControlPlane struct {
	logger logrus.FieldLogger
	inner  *tailscale.Client
}

var _ ControlPlane = (*HostedControlPlane)(nil)

// NewHostedControlPlane creates a new control plane client for the SaaS Tailscale offering
func NewHostedControlPlane(logger logrus.FieldLogger, baseUrl, tailnet string, auth Authentication) (ControlPlane, error) {
	base, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid base url %q: %w", baseUrl, err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("invalid base url scheme %q", base.Scheme)
	}

	client := &tailscale.Client{
		BaseURL: base,
		Tailnet: tailnet,
	}

	if auth != nil {
		auth.tailscale(client)
		logger.WithField("method", auth.Kind()).Debug("applied authentication method")
	}

	logger.
		WithFields(map[string]any{
			"base-url": baseUrl,
			"tailnet":  tailnet,
		}).
		Info("created new hosted control plane client")
	return &HostedControlPlane{logger, client}, nil
}

// Tailnet retrieves the name of the connected tailnet
func (h *HostedControlPlane) Tailnet() string {
	return h.inner.Tailnet
}

// NodeInfo retrieves details about a particular node by its ID
func (h *HostedControlPlane) NodeInfo(ctx context.Context, id string) (*NodeInfo, error) {
	h.logger.WithField("id", id).Debug("fetching node information")
	node, err := h.inner.Devices().Get(ctx, id)
	if err != nil {
		if tailscale.IsNotFound(err) {
			err = nil
		}

		return nil, err
	}

	addresses := make([]netip.Addr, 0, len(node.Addresses))
	for _, raw := range node.Addresses {
		addresses = append(addresses, netip.MustParseAddr(raw))
	}

	return &NodeInfo{
		ID:         node.NodeID,
		Addresses:  addresses,
		Key:        node.NodeKey,
		DNSName:    node.Name,
		Hostname:   node.Hostname,
		Tailnet:    h.inner.Tailnet,
		OS:         node.OS,
		Tags:       node.Tags,
		Authorized: node.Authorized,
		External:   node.IsExternal,
	}, nil
}

// HeadscaleControlPlane connects to a self-hosted Headscale control plane
type HeadscaleControlPlane struct {
	logger  logrus.FieldLogger
	tailnet string
	inner   headscale.HeadscaleServiceClient
}

var _ ControlPlane = (*HeadscaleControlPlane)(nil)

// NewHeadscaleControlPlane creates a new control plane client for a self-hosted Headscale instance
func NewHeadscaleControlPlane(logger logrus.FieldLogger, baseUrl, tailnet string, auth Authentication, tlsMode TLSMode) (ControlPlane, error) {
	var transport credentials.TransportCredentials
	switch tlsMode {
	case TLSModeNone:
		transport = insecure.NewCredentials()
	case TLSModeInsecure:
		transport = credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	case TLSModeFull:
		transport = credentials.NewClientTLSFromCert(nil, "")
	default:
		return nil, ErrUnknownTLSMode
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(transport)}

	if auth != nil {
		opts = append(opts, grpc.WithPerRPCCredentials(auth))
		logger.WithField("method", auth.Kind()).Debug("applied authentication method")
	}

	conn, err := grpc.NewClient(baseUrl, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not create client for %q: %w", baseUrl, err)
	}
	client := headscale.NewHeadscaleServiceClient(conn)

	logger.
		WithFields(map[string]any{
			"bsae-url": baseUrl,
			"tailnet":  tailnet,
		}).
		Info("created new headscale control plane client")
	return &HeadscaleControlPlane{logger, tailnet, client}, nil
}

func (h *HeadscaleControlPlane) Tailnet() string {
	return h.tailnet
}

func (h *HeadscaleControlPlane) NodeInfo(ctx context.Context, id string) (*NodeInfo, error) {
	nodeId, err := strconv.ParseInt(id, 10, 64)
	if err != nil || nodeId <= 0 {
		return nil, errors.New("invalid node id")
	}

	h.logger.WithField("id", nodeId).Debug("fetching node information")
	resp, err := h.inner.GetNode(ctx, &headscale.GetNodeRequest{NodeId: uint64(nodeId)})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.Unknown && s.Message() == "record not found" {
				h.logger.Debug("node not found")
				return nil, nil
			}
		}

		return nil, err
	}
	node := resp.Node

	addresses := make([]netip.Addr, 0, len(node.IpAddresses))
	for _, raw := range node.IpAddresses {
		addresses = append(addresses, netip.MustParseAddr(raw))
	}

	tags := make([]string, len(node.ForcedTags)+len(node.ValidTags)+len(node.InvalidTags))
	tags = append(tags, node.ForcedTags...)
	tags = append(tags, node.ValidTags...)
	tags = append(tags, node.InvalidTags...)

	return &NodeInfo{
		ID:         fmt.Sprintf("%d", node.Id),
		Addresses:  addresses,
		Key:        node.NodeKey,
		DNSName:    node.GivenName,
		Hostname:   node.Name,
		Tailnet:    h.tailnet,
		OS:         "unknown",
		Tags:       tags,
		Authorized: true,
		External:   false,
	}, nil
}

type tokenAuth struct {
	token string
}

var _ credentials.PerRPCCredentials = (*tokenAuth)(nil)

func (t *tokenAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + t.token}, nil
}

func (t *tokenAuth) RequireTransportSecurity() bool {
	return false
}
