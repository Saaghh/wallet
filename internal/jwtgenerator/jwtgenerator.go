package jwtgenerator

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/Saaghh/wallet/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type JWTGenerator struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func NewJWTGenerator() *JWTGenerator {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		zap.L().With(zap.Error(err)).Warn("NewJwtGenerator()/rsa.GenerateKey(new(rand.Rand), 4096)")
	}

	generator := &JWTGenerator{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}

	return generator
}

func (j *JWTGenerator) GetNewTokenString(user model.User) (string, error) {
	claims := model.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "wallet auth server",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UUID: user.ID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)

	ss, err := token.SignedString(j.privateKey)
	if err != nil {
		return "", fmt.Errorf("token.SignedString(j.privateKey): %w", err)
	}

	return ss, nil
}

func (j *JWTGenerator) GetPublicKey() *rsa.PublicKey {
	key := *j.publicKey

	return &key
}
