package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"lib/jwt/model"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"

	cryptoRand "crypto/rand"
)

var letters = []rune(os.Getenv("SECRET_JWT"))

type (
	// JWT
	JWTAuthentication interface {
		GenerateToken(ctx context.Context, userID uint32, domain string) (string, error)
		VerifyToken(ctx context.Context, token, uid, key string) (model.JWTToken, error)
	}

	jwtAuthentication struct {
		expiredAt      int
		signingMethod  jwt.SigningMethod
		publicKey      *rsa.PublicKey
		privateKey     *rsa.PrivateKey
		isRefreshToken bool
		issuer         string
	}
)

func NewJWTVerifyAuthentication(publicKeyStr, signingMethod, issuer string, expiredJWT int, isRefreshToken bool) JWTAuthentication {
	publicStr, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		zap.S().Panic("Failed to parse public key for JWT", zap.Error(err))
	}
	publicKeyFromPEM, err := jwt.ParseRSAPublicKeyFromPEM(publicStr)
	if err != nil {
		zap.S().Panic("Failed to parse public key from PEM for JWT", zap.Error(err))
	}
	return &jwtAuthentication{
		publicKey:      publicKeyFromPEM,
		signingMethod:  jwt.GetSigningMethod(signingMethod),
		expiredAt:      expiredJWT,
		isRefreshToken: isRefreshToken,
		issuer:         issuer,
	}
}

func NewJWTAuthentication(privateKeyStr, publicKeyStr, signingMethod, issuer string, expiredJWT int, isRefreshToken bool) JWTAuthentication {
	privateStr, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err != nil {
		zap.S().Panic("Failed to parse private key for JWT", zap.Error(err))
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateStr)
	if err != nil {
		zap.S().Panic("Failed to parse private key from PEM for JWT", zap.Error(err))
	}
	publicStr, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		zap.S().Panic("Failed to parse public key for JWT", zap.Error(err))
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicStr)
	if err != nil {
		zap.S().Panic("Failed to parse public key from PEM for JWT", zap.Error(err))
	}
	return &jwtAuthentication{
		signingMethod:  jwt.GetSigningMethod(signingMethod),
		privateKey:     privateKey,
		publicKey:      publicKey,
		expiredAt:      expiredJWT,
		isRefreshToken: isRefreshToken,
		issuer:         issuer,
	}
}

func randString(n int) (str string) {
	b := make([]rune, n)
	for i := range b {
		randIndex, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return str
		}
		b[i] = letters[randIndex.Int64()]
	}
	return string(b)
}

func getRefreshToken() string {
	dest, _ := hex.DecodeString(fmt.Sprintf("%d", time.Now().UnixNano()))
	var id strings.Builder
	encode := base64.StdEncoding.EncodeToString(dest)
	rand.Seed(time.Now().UnixNano())
	id.WriteString(encode)
	id.WriteString(randString(4))
	return strings.Replace(id.String(), "=", randString(1), 1)
}

func (j *jwtAuthentication) GenerateToken(ctx context.Context, userID uint32, domain string) (token string, err error) {

	jwtClaim := jwt.New(j.signingMethod)
	var claim model.JWTToken
	if j.isRefreshToken {
		claim = model.JWTToken{
			Domain:       domain,
			UserID:       userID,
			RefreshToken: getRefreshToken(),
			StandardClaims: &jwt.StandardClaims{
				Issuer:    j.issuer,
				ExpiresAt: time.Now().Add(time.Second * time.Duration(j.expiredAt)).Unix(),
			},
		}
	} else {
		claim = model.JWTToken{
			Domain: domain,
			UserID: userID,
			StandardClaims: &jwt.StandardClaims{
				Issuer:    j.issuer,
				ExpiresAt: time.Now().Add(time.Second * time.Duration(j.expiredAt)).Unix(),
			},
		}
	}

	jwtClaim.Claims = claim

	token, err = jwtClaim.SignedString(j.privateKey)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (j *jwtAuthentication) VerifyToken(ctx context.Context, tokenStr, uid, key string) (model.JWTToken, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return j.publicKey, nil
	}

	var claim model.JWTToken
	token, err := jwt.ParseWithClaims(tokenStr, &claim, keyFunc)
	v, _ := err.(*jwt.ValidationError)
	if v != nil && v.Errors == jwt.ValidationErrorExpired && claim.RefreshToken != "" {
		err = errors.New("JWT is expired")
		return claim, err
	}
	if err != nil || !token.Valid {
		err = errors.New("JWT not valid")
		return claim, err
	}

	return claim, nil
}
