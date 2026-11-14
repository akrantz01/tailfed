package types

import "github.com/go-jose/go-jose/v4/jwt"

type TokenResponse struct {
	TokenType   string          `json:"token_type"`
	AccessToken string          `json:"access_token"`
	ExpiresIn   float64         `json:"expires_in"`
	ExpiresAt   jwt.NumericDate `json:"expires_at"`
}
