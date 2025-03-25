package types

// APIResponse is the general structure of the HTTP response payload
type APIResponse[T any] struct {
	// Success signifies whether the response was a success
	Success bool `json:"success"`
	// Data is the endpoint-specific response data, only present when Success is `true`
	Data *T `json:"data,omitempty"`
	// Error is a description of what went wrong, only present when Success is `false`
	Error *Error `json:"error,omitempty"`
}

// Error represents a server-side error
type Error struct {
	// Message a short message describing what went wrong
	Message string `json:"message"`
	// Retryable whether the request should be retried
	Retryable bool `json:"retryable"`
}

// StartRequest is sent by the client to initiate a token issuance flow
type StartRequest struct {
	// Node contains the ID of the Tailscale node
	Node string `json:"node"`
	// Port contains the port the node is listening on in the tailnet
	Port int `json:"port"`
}

// StartResponse is returned by the start handler
type StartResponse struct {
	// ID is a unique identifier for the challenge
	ID string `json:"id"`
	// Token is a signing secret used to generate a HMAC-SHA256
	Token string `json:"token"`
}

// ChallengeResponse is returned by the client challenge handler
type ChallengeResponse struct {
	// Signature is a HMAC-SHA256 of the node key and tailnet IPs (sorted lexicographically)
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
