package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apporders "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/orders"
)

func TestGatewayAuthenticatorRejectsEmptyBearerBeforeTrustedHeaders(t *testing.T) {
	authenticator, err := NewGatewayAuthenticator("local-gateway-token", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer ")
	headers.Set("X-StreamWeave-Actor-ID", "usr_local")
	headers.Set("X-StreamWeave-Actor-Type", "human")
	headers.Set("X-StreamWeave-Scopes", "orders:write:self")

	_, err = authenticator.Authenticate(context.Background(), headers)
	if !errors.Is(err, apporders.ErrUnauthorized) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestGatewayAuthenticatorRequiresStaticTokenAndValidActorContext(t *testing.T) {
	authenticator, err := NewGatewayAuthenticator("local-gateway-token", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer local-gateway-token")
	headers.Set("X-StreamWeave-Actor-ID", "usr_local")
	headers.Set("X-StreamWeave-Actor-Type", "human")
	headers.Set("X-StreamWeave-Scopes", "orders:read orders:write:self")

	actor, err := authenticator.Authenticate(context.Background(), headers)
	if err != nil {
		t.Fatal(err)
	}
	if actor.Subject != "usr_local" || actor.Type != "human" || len(actor.Scopes) != 2 {
		t.Fatalf("unexpected actor: %#v", actor)
	}

	headers.Set("Authorization", "Bearer wrong-token")
	if _, err := authenticator.Authenticate(context.Background(), headers); !errors.Is(err, apporders.ErrUnauthorized) {
		t.Fatalf("expected invalid token to be rejected, got %v", err)
	}
}
