package oidc

import (
	"reflect"
	"strings"

	"github.com/go-jose/go-jose/v4/jwt"
)

// Claims contains the data that will be signed in the token
type Claims struct {
	AuthenticatedMethodsReference []string `json:"amr"`
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
