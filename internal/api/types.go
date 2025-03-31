package api

import "net/netip"

// Response is the general structure of the HTTP response payload
type Response[T any] struct {
	// Success signifies whether the response was a success
	Success bool `json:"success"`
	// Data is the endpoint-specific response data, only present when Success is `true`
	Data *T `json:"data,omitempty"`
	// Error is a description of what went wrong, only present when Success is `false`
	Error string `json:"error,omitempty"`
}

// StartRequest is sent by the client to initiate a token issuance flow
type StartRequest struct {
	// Node contains the ID of the Tailscale node
	Node string `json:"node"`
	// PortBindings contain the ports the node is listening on in the tailnet
	PortBindings []PortBinding `json:"port-bindings"`
}

// Network represents the kind of network the port is bound to
type Network string

var (
	NetworkV4      Network = "v4"
	NetworkV6      Network = "v6"
	NetworkUnknown Network = "unknown"
)

// NetworkFromAddrPort creates a Network from a netip.AddrPort
func NetworkFromAddrPort(addr netip.AddrPort) Network {
	if addr.Addr().Is6() {
		return NetworkV6
	} else if addr.Addr().Is4() {
		return NetworkV4
	} else {
		return NetworkUnknown
	}
}

// Valid checks if the Network is valid
func (a Network) Valid() bool {
	return a == NetworkV4 || a == NetworkV6
}

// PortBinding represents a listening port on a particular network type
type PortBinding struct {
	// Port is the system port being listened on
	Port uint16 `json:"port"`
	// Network is the type of address the port is listening on
	Network Network `json:"network"`
}

// StartResponse is returned by the start handler
type StartResponse struct {
	// ID is a unique identifier for the challenge
	ID string `json:"id"`
	// SigningSecret is used to generate a HMAC-SHA256 signature of the client details
	SigningSecret []byte `json:"signing-secret"`
}

// ChallengeResponse is returned by the client challenge handler
type ChallengeResponse struct {
	// Signature is a HMAC-SHA256 of the tailnet, hostname, node key and operating system
	Signature string `json:"signature"`
}

// FinalizeRequest is sent by the client once the challenge has been sent
type FinalizeRequest struct {
	// ID is the unique identifier for the challenge from StartRequest
	ID string `json:"id"`
}

// FinalizeResponse is sent by the finalize handler once the challenge has been successfully authenticated
type FinalizeResponse struct {
	// IdentityToken is a signed JWT that can be used to generate AWS credentials
	IdentityToken string `json:"identity-token"`
}
