package api

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
