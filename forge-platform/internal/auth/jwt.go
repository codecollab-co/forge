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
	pub  any
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
	s := &Signer{key: jwt.SigningMethodRS256, priv: priv}
	if pubRaw := os.Getenv("JWT_PUBLIC_KEY"); strings.TrimSpace(pubRaw) != "" {
		pubPem := strings.ReplaceAll(pubRaw, `\n`, "\n")
		pub, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pubPem))
		if err != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		s.pub = pub
	}
	return s, nil
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

// Verify returns the subject of a token issued by this Signer (or another
// service holding the same key pair). Used to authenticate service-to-service
// calls on /internal/* endpoints.
func (s *Signer) Verify(token string) (string, error) {
	if s.pub == nil {
		return "", errors.New("verifier has no public key")
	}
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.pub, nil
	}, jwt.WithIssuer("forge-platform"))
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	sub, _ := claims["sub"].(string)
	return sub, nil
}
