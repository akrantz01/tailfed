package types

import "net/netip"

// StartRequest is sent by the client to initiate a token issuance flow
type StartRequest struct {
	// Node contains the ID of the Tailscale node
	Node string `json:"node"`
	// Ports contains the listening ports for the tailnet addresses
	Ports Ports `json:"ports"`
}

// Ports contains the listening ports for the IPv4 and IPv6 tailnet addresses
type Ports struct {
	// IPv4 contains the listening port for the v4 address
	IPv4 uint16 `json:"ipv4"`
	// IPv6 contains the listening port for the v6 address
	IPv6 uint16 `json:"ipv6"`
}

// VerifyRequest is sent by the launcher backend to perform the challenge verification
type VerifyRequest struct {
	// ID contains the flow's unique identifier
	ID string `json:"id"`
	// Address contains the IP address-port pairs to test
	Address netip.AddrPort `json:"address"`
}

// FinalizeRequest is sent by the client once the challenge has been sent
type FinalizeRequest struct {
	// ID is the unique identifier for the challenge from StartRequest
	ID string `json:"id"`
}
