//go:build security

package security

import (
	"strings"
	"testing"
	"time"

	identitycrypto "github.com/chouaib-skitou/streamweave/apps/identity/internal/adapters/outbound/crypto"
)

func TestAccessTokenVerificationRejectsForgedExpiredAndWrongAudienceTokens(t *testing.T) {
	signer, err := identitycrypto.GenerateSigner("security-test", "https://identity.test")
	if err != nil {
		t.Fatal(err)
	}
	token, err := signer.Sign("usr_security", "platform-api", "human", "session-security", nil, []string{"orders:read:self"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.Verify(token, "wrong-audience"); err == nil {
		t.Fatal("token with the wrong audience was accepted")
	}
	parts := strings.Split(token, ".")
	parts[2] = strings.Repeat("A", len(parts[2]))
	if _, err := signer.Verify(strings.Join(parts, "."), "platform-api"); err == nil {
		t.Fatal("forged token was accepted")
	}
	expired, err := signer.Sign("usr_security", "platform-api", "human", "session-security", nil, nil, -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.Verify(expired, "platform-api"); err == nil {
		t.Fatal("expired token was accepted")
	}
}
