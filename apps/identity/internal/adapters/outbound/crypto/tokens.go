package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type TokenService struct{ signer *Signer }

func NewTokenService() *TokenService {
	// The signer is injected after construction through NewTokenServiceForSigner.
	return &TokenService{}
}

func NewTokenServiceForSigner(signer *Signer) *TokenService { return &TokenService{signer: signer} }

func (t *TokenService) NewOpaque(size int) (string, []byte, error) {
	if size < 32 {
		return "", nil, fmt.Errorf("opaque token size must be at least 32 bytes")
	}
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate opaque token: %w", err)
	}
	value := hex.EncodeToString(raw)
	return value, t.HashOpaque(value), nil
}

func (t *TokenService) HashOpaque(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func (t *TokenService) Sign(subject, audience, kind, sessionID string, roles, scopes []string, ttl time.Duration) (string, error) {
	if t.signer == nil {
		return "", fmt.Errorf("token signer is not configured")
	}
	return t.signer.Sign(subject, audience, kind, sessionID, roles, scopes, ttl)
}
