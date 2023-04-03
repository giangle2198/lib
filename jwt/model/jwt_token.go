package model

import "github.com/dgrijalva/jwt-go"

// JWTTokenClaim
type JWTToken struct {
	*jwt.StandardClaims
	UserID       string
	RefreshToken string
	Domain       string
}
