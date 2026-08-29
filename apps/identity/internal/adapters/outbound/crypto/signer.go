package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Roles     []string `json:"roles,omitempty"`
	Scopes    []string `json:"scp,omitempty"`
	TokenKind string   `json:"token_kind"`
	SessionID string   `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	kid        string
	issuer     string
}

func NewSigner(privateKey ed25519.PrivateKey, kid, issuer string) (*Signer, error) {
	if len(privateKey) != ed25519.PrivateKeySize || kid == "" || issuer == "" {
		return nil, fmt.Errorf("invalid signing configuration")
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return &Signer{privateKey: privateKey, publicKey: publicKey, kid: kid, issuer: issuer}, nil
}

func GenerateSigner(kid, issuer string) (*Signer, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return NewSigner(privateKey, kid, issuer)
}

func LoadSigner(path, kid, issuer string) (*Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read signing key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode signing key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse signing key: %w", err)
	}
	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("signing key is not Ed25519")
	}
	return NewSigner(privateKey, kid, issuer)
}

func (s *Signer) Sign(subject, audience, kind, sessionID string, roles, scopes []string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{Roles: roles, Scopes: scopes, TokenKind: kind, SessionID: sessionID, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: s.issuer, Subject: subject, Audience: jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), IssuedAt: jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now), ID: uuid.NewString(),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = s.kid
	return token.SignedString(s.privateKey)
}

func (s *Signer) Verify(raw, audience string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodEdDSA {
			return nil, fmt.Errorf("unexpected signing algorithm")
		}
		if kid, ok := token.Header["kid"].(string); !ok || kid != s.kid {
			return nil, fmt.Errorf("unknown signing key")
		}
		return s.publicKey, nil
	}, jwt.WithAudience(audience), jwt.WithIssuer(s.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}))
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid access token")
	}
	return claims, nil
}

func (s *Signer) PublicKey() ed25519.PublicKey { return s.publicKey }
func (s *Signer) KeyID() string                { return s.kid }
func (s *Signer) Issuer() string               { return s.issuer }

func (s *Signer) JWKS() map[string]any {
	return map[string]any{"keys": []map[string]any{{
		"kty": "OKP", "crv": "Ed25519", "kid": s.kid, "use": "sig", "alg": "EdDSA",
		"x": base64.RawURLEncoding.EncodeToString(s.publicKey),
	}}}
}
