package types

// Response is the general structure of the HTTP response payload
type Response[T any] struct {
	// Success signifies whether the response was a success
	Success bool `json:"success"`
	// Data is the endpoint-specific response data, only present when Success is `true`
	Data *T `json:"data,omitempty"`
	// Error is a description of what went wrong, only present when Success is `false`
	Error string `json:"error,omitempty"`
}

// ConfigResponse provides configuration to the daemon
type ConfigResponse struct {
	// Frequency determines how often the token should be refreshed
	Frequency Duration `json:"frequency"`
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
	Signature []byte `json:"signature"`
}

// VerifyResponse is returned by the verifier handler whenever an attempt completes
type VerifyResponse struct {
	// Success denotes whether the verification was successful
	Success bool `json:"success"`
}

// FinalizeResponse is sent by the finalize handler once the challenge has been successfully authenticated
type FinalizeResponse struct {
	// IdentityToken is a signed JWT that can be used to generate AWS credentials
	IdentityToken string `json:"identity-token"`
}
