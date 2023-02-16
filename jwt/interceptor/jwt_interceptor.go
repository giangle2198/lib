package interceptor

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"lib/common"
	"lib/jwt/model"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
)

type (
	// JWTAdapter
	JWTAdapter interface {
		GenerateToken(ctx context.Context, userID uint32, domain string) (a, b string, c error)
		VerifyToken(ctx context.Context, token, uid, key string) (model.JWTToken, error)
	}

	jwtAdapter struct {
		signingMethod       jwt.SigningMethod
		publicKey           *rsa.PublicKey
		privateKey          *rsa.PrivateKey
		issuer              string
		isUsageRefreshToken bool
		tokenExpireTime     time.Duration
	}
)

// NewJWTVerifyAdapter
func NewJWTVerifyAdapter(publicKeyStr, signingMethod string) JWTAdapter {
	publicStr, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		zap.S().Panicf("Error at init public key for JWT: %v", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicStr)
	if err != nil {
		zap.S().Panicf("Error at init public key for JWT: %v", err)
	}
	return &jwtAdapter{
		publicKey:     publicKey,
		signingMethod: jwt.GetSigningMethod(signingMethod),
	}
}

// NewJWTAdapter
func NewJWTAdapter(issuer, signingMethod, publicKey, privateKey string, isUsageRefreshToken bool, tokenExpireTime time.Duration) JWTAdapter {
	privateStr, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		zap.S().Panicf("Error at init private key for JWT: %v", err)
	}
	prvKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateStr)
	if err != nil {
		zap.S().Panicf("Error at init private key for JWT: %v", err)
	}
	publicStr, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		zap.S().Panicf("Error at init public key for JWT: %v", err)
	}
	pbKey, err := jwt.ParseRSAPublicKeyFromPEM(publicStr)
	if err != nil {
		zap.S().Panicf("Error at init public key for JWT: %v", err)
	}
	return &jwtAdapter{
		signingMethod:       jwt.GetSigningMethod(signingMethod),
		privateKey:          prvKey,
		publicKey:           pbKey,
		issuer:              issuer,
		isUsageRefreshToken: isUsageRefreshToken,
		tokenExpireTime:     tokenExpireTime,
	}
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

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

func nowAsUnixSecond() int64 {
	return time.Now().UnixNano() / 1e9
}

func getRefreshToken() string {
	dest, _ := hex.DecodeString(fmt.Sprintf("%d", nowAsUnixSecond()))
	var id strings.Builder
	encode := base64.StdEncoding.EncodeToString(dest)
	rand.Seed(time.Now().UnixNano())
	id.WriteString(encode)
	id.WriteString(randString(4))
	return strings.Replace(id.String(), "=", randString(1), 1)
}

// GenerateToken generate new token
func (j *jwtAdapter) GenerateToken(ctx context.Context, userID uint32, domain string) (a, b string, c error) {
	token := jwt.New(j.signingMethod)
	var claim model.JWTToken

	if j.isUsageRefreshToken {
		refreshToken := getRefreshToken()
		claim = model.JWTToken{
			UserID:       userID,
			RefreshToken: refreshToken,
			Domain:       domain,
			StandardClaims: &jwt.StandardClaims{
				Issuer:    j.issuer,
				ExpiresAt: time.Now().Add(time.Second * time.Duration(j.tokenExpireTime)).Unix(),
			},
		}
	}

	if !j.isUsageRefreshToken {
		claim = model.JWTToken{
			UserID: userID,
			Domain: domain,
			StandardClaims: &jwt.StandardClaims{
				Issuer:    j.issuer,
				ExpiresAt: time.Now().Add(time.Second * time.Duration(j.tokenExpireTime)).Unix(),
			},
		}
	}

	token.Claims = claim

	tokenStr, err := token.SignedString(j.privateKey)
	if err != nil {
		return "", "", err
	}
	return tokenStr, claim.RefreshToken, nil
}

func (j *jwtAdapter) VerifyToken(ctx context.Context, tokenStr, uid, key string) (model.JWTToken, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return j.publicKey, nil
	}
	var claim model.JWTToken
	token, err := jwt.ParseWithClaims(tokenStr, &claim, keyFunc)
	v, _ := err.(*jwt.ValidationError)
	if v != nil && v.Errors == jwt.ValidationErrorExpired && claim.RefreshToken != "" {
		err = errors.New(common.ReasonJWTExpired.Code())
		return claim, err
	}
	if err != nil || !token.Valid {
		err = errors.New(common.ReasonJWTInvalid.Code())
		return claim, err
	}
	return claim, nil
}
