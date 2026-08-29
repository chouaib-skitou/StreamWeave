package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func NewOpaqueToken(size int) (string, []byte, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], nil
}

func HashOpaqueToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}
