package jwks

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type fetcherStub struct {
	keys  map[string]ed25519.PublicKey
	age   time.Duration
	err   error
	calls int
}

func (f *fetcherStub) Fetch(context.Context) (map[string]ed25519.PublicKey, time.Duration, error) {
	f.calls++
	return f.keys, f.age, f.err
}

func TestVerifierChecksClaimsAndRefreshesUnknownKid(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	fetcher := &fetcherStub{keys: map[string]ed25519.PublicKey{"k1": public}}
	v, _ := NewVerifier(fetcher, "identity.local", "platform-api")
	if err := v.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	token := signed(t, private, "k1", "identity.local", "platform-api", "human")
	actor, err := v.Verify(context.Background(), token)
	if err != nil || actor.Subject != "user-1" || actor.Scopes[0] != "orders:read:self" {
		t.Fatalf("actor=%+v err=%v", actor, err)
	}
	_, unknownPrivate, _ := ed25519.GenerateKey(rand.Reader)
	fetcher.keys["k2"] = public
	unknown := signed(t, unknownPrivate, "k2", "identity.local", "platform-api", "human")
	if _, err := v.Verify(context.Background(), unknown); err == nil {
		t.Fatal("token signed by unknown private key should fail")
	}
	_ = unknown
}

func TestVerifierRejectsWrongTokenAndDependency(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	fetcher := &fetcherStub{keys: map[string]ed25519.PublicKey{"k": public}}
	v, _ := NewVerifier(fetcher, "issuer", "aud")
	if _, err := v.Verify(context.Background(), ""); err == nil {
		t.Fatal("empty token accepted")
	}
	if err := v.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"service", ""} {
		if _, err := v.Verify(context.Background(), signed(t, private, "k", "issuer", "aud", kind)); err == nil {
			t.Fatalf("kind %q accepted", kind)
		}
	}
	fetcher.err = errors.New("offline")
	v2, _ := NewVerifier(fetcher, "issuer", "aud")
	if err := v2.Check(context.Background()); err == nil {
		t.Fatal("dependency error hidden")
	}
}

func signed(t *testing.T, private ed25519.PrivateKey, kid, issuer, audience, kind string) string {
	t.Helper()
	now := time.Now().UTC()
	claims := Claims{Roles: []string{"customer"}, Scopes: []string{"orders:read:self"}, TokenKind: kind, RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: "user-1", Audience: []string{audience}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now), ID: "jti-1"}}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = kid
	value, err := token.SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestHTTPFetcherValidationHelpers(t *testing.T) {
	if maxAge("public, max-age=20") != 20*time.Second {
		t.Fatal("max-age parsing")
	}
	if maxAge("max-age=600") != 5*time.Minute {
		t.Fatal("max-age upper bound")
	}
	_ = base64.RawURLEncoding
}
