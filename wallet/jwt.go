package wallet

import "github.com/golang-jwt/jwt/v5"

// JWTClaim represents the claims in the JWT
type JWTClaim struct {
	Login string `json:"login"`
	jwt.RegisteredClaims
}
