package signing

import (
	"time"

	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Backend provides a mechanism for generating signed JWTs
type Backend interface {
	// PublicKey returns details about the public key
	PublicKey() (jose.JSONWebKey, error)
	// Sign generates a signed JWT with the provided claims
	Sign(claims Claims) (string, error)
}

// Claims contains the data that will be signed in the token
type Claims struct {
	jwt.Claims
	Tailnet     string   `json:"tailnet"`
	DNSName     string   `json:"dns_name"`
	MachineName string   `json:"machine_name"`
	HostName    string   `json:"host_name"`
	OS          string   `json:"os"`
	Tags        []string `json:"tags"`
	Authorized  bool     `json:"authorized"`
	External    bool     `json:"external"`
}

// NewClaimsFromFlow creates a new claim set from an in-progress flow
func NewClaimsFromFlow(issuer, audience string, validity time.Duration, flow *storage.Flow) Claims {
	now := time.Now()
	nowNumeric := jwt.NewNumericDate(now)

	return Claims{
		Claims: jwt.Claims{
			Issuer:    issuer,
			Audience:  jwt.Audience{audience},
			Subject:   flow.Node,
			IssuedAt:  nowNumeric,
			NotBefore: nowNumeric,
			Expiry:    jwt.NewNumericDate(now.Add(validity)),
		},
		Tailnet:     flow.Tailnet,
		DNSName:     flow.DNSName,
		MachineName: flow.MachineName,
		HostName:    flow.Hostname,
		OS:          flow.OS,
		Tags:        flow.Tags,
		Authorized:  flow.Authorized,
		External:    flow.External,
	}
}

// newKey creates a new [jose.Signer] for generating new JWTs with an embedded key ID in the header
func newKey(keyId string, key any, algorithm jose.SignatureAlgorithm) (jose.Signer, error) {
	return jose.NewSigner(
		jose.SigningKey{Key: key, Algorithm: algorithm},
		&jose.SignerOptions{
			ExtraHeaders: map[jose.HeaderKey]any{"kid": keyId},
		},
	)
}
