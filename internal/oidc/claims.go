package oidc

import (
	"reflect"
	"strings"
	"time"

	"github.com/akrantz01/tailfed/internal/storage"
	"github.com/go-jose/go-jose/v4/jwt"
)

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

func claimsKeys() []string {
	t := reflect.TypeFor[Claims]()
	return structKeys(t)
}

func structKeys(t reflect.Type) []string {
	var keys []string
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous {
			keys = append(keys, structKeys(f.Type)...)
		} else if value, ok := f.Tag.Lookup("json"); ok {
			parts := strings.Split(value, ",")
			keys = append(keys, parts[0])
		}
	}

	return keys
}
