// Package auth handles JWT issuance for forge-platform.
//
// RS256 with the private key supplied via the JWT_PRIVATE_KEY env var (PEM,
// with literal \n escapes for newlines). Downstream services verify with the
// matching public key from JWT_PUBLIC_KEY.
package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Signer struct {
	key *jwt.SigningMethodRSA
	priv any
}

func NewSignerFromEnv() (*Signer, error) {
	raw := os.Getenv("JWT_PRIVATE_KEY")
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("JWT_PRIVATE_KEY is empty")
	}
	pem := strings.ReplaceAll(raw, `\n`, "\n")
	priv, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pem))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return &Signer{key: jwt.SigningMethodRS256, priv: priv}, nil
}

func (s *Signer) Issue(subject string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    "forge-platform",
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	tok := jwt.NewWithClaims(s.key, claims)
	return tok.SignedString(s.priv)
}
