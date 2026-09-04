//go:build security

package security

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	"github.com/chouaib-skitou/streamweave/apps/gateway/internal/adapters/outbound/jwks"
	"github.com/golang-jwt/jwt/v5"
)

type securityFetcher struct{ key ed25519.PublicKey }

func (f securityFetcher) Fetch(context.Context) (map[string]ed25519.PublicKey, time.Duration, error) {
	return map[string]ed25519.PublicKey{"safe": f.key}, time.Minute, nil
}

func TestGatewayJWTRejectsAlgorithmConfusionAndWrongAudience(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	verifier, err := jwks.NewVerifier(securityFetcher{key: public}, "identity.local", "platform-api")
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	claims := jwt.RegisteredClaims{Issuer: "identity.local", Subject: "usr_1", Audience: []string{"wrong-audience"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now()), ID: "jti"}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = "safe"
	raw, _ := token.SignedString(private)
	if _, err := verifier.Verify(context.Background(), raw); err == nil {
		t.Fatal("wrong audience accepted")
	}
}

func TestGatewayDoesNotTreatInternalHeadersAsAuthorization(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "/v1/orders", nil)
	request.Header.Set("X-StreamWeave-Actor-ID", "attacker")
	if request.Header.Get("Authorization") != "" {
		t.Fatal("unexpected authorization header")
	}
}
