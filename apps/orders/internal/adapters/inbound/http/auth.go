package httpadapter

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	apporders "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/orders"
	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
	"github.com/golang-jwt/jwt/v5"
)

type GatewayAuthenticator struct {
	staticToken string
	jwksURL     string
	issuer      string
	audience    string
	subject     string
	client      *http.Client
	mu          sync.Mutex
	keys        map[string]ed25519.PublicKey
	loadedAt    time.Time
}

func NewGatewayAuthenticator(staticToken, jwksURL, issuer, audience, subject string) (*GatewayAuthenticator, error) {
	if strings.TrimSpace(staticToken) == "" && strings.TrimSpace(jwksURL) == "" {
		return nil, errors.New("Orders Gateway authentication requires a static token or JWKS URL")
	}
	return &GatewayAuthenticator{staticToken: staticToken, jwksURL: jwksURL, issuer: issuer, audience: audience, subject: subject, client: &http.Client{Timeout: 2 * time.Second}}, nil
}

func (a *GatewayAuthenticator) Authenticate(ctx context.Context, headers http.Header) (apporders.Actor, error) {
	values := headers.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return apporders.Actor{}, apporders.ErrUnauthorized
	}
	raw := strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer "))
	if raw == "" || (a.staticToken != "" && raw == a.staticToken) {
		return trustedActor(headers)
	}
	if a.staticToken != "" {
		return apporders.Actor{}, apporders.ErrUnauthorized
	}
	claims := &machineClaims{}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}), jwt.WithIssuer(a.issuer), jwt.WithAudience(a.audience), jwt.WithLeeway(30*time.Second))
	token, err := parser.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("missing key id")
		}
		key, ok := a.key(ctx, kid)
		if !ok {
			return nil, errors.New("unknown key id")
		}
		return key, nil
	})
	if err != nil || token == nil || !token.Valid || claims.TokenKind != "service" || claims.Subject != a.subject || claims.ExpiresAt == nil || claims.ID == "" || !hasOrdersMachineScope(claims.Scopes) {
		return apporders.Actor{}, apporders.ErrUnauthorized
	}
	return trustedActor(headers)
}

func hasOrdersMachineScope(scopes []string) bool {
	for _, scope := range scopes {
		if scope == "orders:read" || scope == "orders:write" {
			return true
		}
	}
	return false
}

type machineClaims struct {
	TokenKind string   `json:"token_kind"`
	Scopes    []string `json:"scp,omitempty"`
	jwt.RegisteredClaims
}

func trustedActor(headers http.Header) (apporders.Actor, error) {
	subject := strings.TrimSpace(headers.Get("X-StreamWeave-Actor-ID"))
	if !domain.ValidCustomerID(subject) || strings.TrimSpace(headers.Get("X-StreamWeave-Actor-Type")) != "human" {
		return apporders.Actor{}, apporders.ErrUnauthorized
	}
	scopes := strings.Fields(headers.Get("X-StreamWeave-Scopes"))
	for _, scope := range scopes {
		if !scopePattern.MatchString(scope) {
			return apporders.Actor{}, apporders.ErrUnauthorized
		}
	}
	return apporders.Actor{Subject: subject, Type: "human", Scopes: scopes}, nil
}

func (a *GatewayAuthenticator) key(ctx context.Context, kid string) (ed25519.PublicKey, bool) {
	a.mu.Lock()
	if key, ok := a.keys[kid]; ok && time.Since(a.loadedAt) < 5*time.Minute {
		a.mu.Unlock()
		return key, true
	}
	a.mu.Unlock()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, a.jwksURL, nil)
	if err != nil {
		return nil, false
	}
	response, err := a.client.Do(request)
	if err != nil {
		return nil, false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, false
	}
	var document struct {
		Keys []struct{ KTY, CRV, KID, Use, Alg, X string } `json:"keys"`
	}
	if json.Unmarshal(body, &document) != nil {
		return nil, false
	}
	keys := make(map[string]ed25519.PublicKey, len(document.Keys))
	for _, item := range document.Keys {
		if item.KTY != "OKP" || item.CRV != "Ed25519" || item.Use != "sig" || item.Alg != "EdDSA" {
			continue
		}
		decoded, decodeErr := base64.RawURLEncoding.DecodeString(item.X)
		if decodeErr == nil && len(decoded) == ed25519.PublicKeySize {
			keys[item.KID] = ed25519.PublicKey(decoded)
		}
	}
	a.mu.Lock()
	a.keys = keys
	a.loadedAt = time.Now()
	key, ok := keys[kid]
	a.mu.Unlock()
	return key, ok
}
