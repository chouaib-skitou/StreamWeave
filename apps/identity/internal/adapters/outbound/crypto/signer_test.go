package crypto

import (
	"testing"
	"time"
)

func TestSignerRejectsWrongAudienceAndPreservesClaims(t *testing.T) {
	signer, err := GenerateSigner("kid-1", "identity.test")
	if err != nil {
		t.Fatal(err)
	}
	token, err := signer.Sign("usr_123", "platform-api", "human", "session-1", []string{"customer"}, []string{"orders:read:self"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := signer.Verify(token, "platform-api")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "usr_123" || claims.SessionID != "session-1" || claims.TokenKind != "human" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := signer.Verify(token, "wrong-audience"); err == nil {
		t.Fatal("wrong audience accepted")
	}
	if len(signer.JWKS()["keys"].([]map[string]any)) != 1 {
		t.Fatal("expected one public key")
	}
}
