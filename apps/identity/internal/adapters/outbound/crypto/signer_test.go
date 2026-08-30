package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
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

func TestSignerAcceptsConfiguredKeyHistoryAndPublishesDeterministicJWKS(t *testing.T) {
	oldSigner, err := GenerateSigner("identity-v1", "identity.test")
	if err != nil {
		t.Fatal(err)
	}
	oldToken, err := oldSigner.Sign("usr_old", "platform-api", "human", "session", nil, nil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	_, activeKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	temporary := t.TempDir()
	activeBytes, err := x509.MarshalPKCS8PrivateKey(activeKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temporary, "active.pem"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: activeBytes}), 0600); err != nil {
		t.Fatal(err)
	}
	history := filepath.Join(temporary, "history")
	if err := os.Mkdir(history, 0700); err != nil {
		t.Fatal(err)
	}
	oldBytes, err := x509.MarshalPKIXPublicKey(oldSigner.publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(history, "identity-v1.pub.pem"), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: oldBytes}), 0600); err != nil {
		t.Fatal(err)
	}
	rotated, err := LoadSignerWithHistory(filepath.Join(temporary, "active.pem"), "identity-v2", "identity.test", history)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rotated.Verify(oldToken, "platform-api"); err != nil {
		t.Fatalf("history token rejected: %v", err)
	}
	if got := len(rotated.JWKS()["keys"].([]map[string]any)); got != 2 {
		t.Fatalf("JWKS keys: got %d, want 2", got)
	}
}
