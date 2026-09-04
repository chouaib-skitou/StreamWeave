package httpadapter

import (
	"net/http/httptest"
	"testing"
	"time"

	identitycrypto "github.com/chouaib-skitou/streamweave/apps/identity/internal/adapters/outbound/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestGatewayActorClaimsAcceptsOnlyBoundActorContext(t *testing.T) {
	machine := &identitycrypto.Claims{TokenKind: "service", Scopes: []string{"identity:sessions:read"}, RegisteredClaims: jwt.RegisteredClaims{Subject: "svc_gateway", ID: "machine-jti", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}}
	server := &Server{gatewaySubject: "svc_gateway"}
	session := uuid.New()
	request := httptest.NewRequest("GET", "/v1/auth/sessions", nil)
	request.Header.Set("X-StreamWeave-Actor-ID", "usr_00000000000000000000000000000001")
	request.Header.Set("X-StreamWeave-Actor-Type", "human")
	request.Header.Set("X-StreamWeave-Actor-Session-ID", session.String())
	request.Header.Set("X-StreamWeave-Scopes", "identity:sessions:read")
	claims, err := server.gatewayActorClaims(request, machine)
	if err != nil || claims.Subject == machine.Subject || claims.SessionID != session.String() || claims.TokenKind != "service-context" {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
	request.Header.Set("X-StreamWeave-Actor-ID", "attacker")
	if _, err := server.gatewayActorClaims(request, machine); err == nil {
		t.Fatal("invalid actor accepted")
	}
	request.Header.Set("X-StreamWeave-Actor-ID", "usr_00000000000000000000000000000001")
	request.Header.Set("X-StreamWeave-Scopes", "orders:write:any")
	if _, err := server.gatewayActorClaims(request, machine); err == nil {
		t.Fatal("ungranted actor scope accepted")
	}
}
