package jwks

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	domain "github.com/chouaib-skitou/streamweave/apps/gateway/internal/domain/gateway"
	"github.com/golang-jwt/jwt/v5"
)

var ErrNotReady = errors.New("jwks is not ready")

type Fetcher interface {
	Fetch(context.Context) (map[string]ed25519.PublicKey, time.Duration, error)
}

type HTTPFetcher struct {
	URL    string
	Client *http.Client
}

func (f HTTPFetcher) Fetch(ctx context.Context) (map[string]ed25519.PublicKey, time.Duration, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, f.URL, nil)
	if err != nil {
		return nil, 0, err
	}
	response, err := f.Client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("jwks status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, 0, err
	}
	var document struct {
		Keys []struct{ KTY, CRV, KID, Use, Alg, X string } `json:"keys"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, 0, err
	}
	keys := make(map[string]ed25519.PublicKey, len(document.Keys))
	for _, key := range document.Keys {
		if key.KTY != "OKP" || key.CRV != "Ed25519" || key.Alg != "EdDSA" || key.Use != "sig" || strings.TrimSpace(key.KID) == "" {
			return nil, 0, errors.New("invalid jwks key metadata")
		}
		decoded, err := base64.RawURLEncoding.DecodeString(key.X)
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			return nil, 0, errors.New("invalid jwks public key")
		}
		keys[key.KID] = ed25519.PublicKey(decoded)
	}
	if len(keys) == 0 {
		return nil, 0, errors.New("jwks contains no keys")
	}
	return keys, maxAge(response.Header.Get("Cache-Control")), nil
}

func maxAge(value string) time.Duration {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			seconds, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
			if err == nil && seconds > 0 {
				d := time.Duration(seconds) * time.Second
				if d < 5*time.Minute {
					return d
				}
				return 5 * time.Minute
			}
		}
	}
	return 5 * time.Minute
}

type Claims struct {
	Roles     []string `json:"roles,omitempty"`
	Scopes    []string `json:"scp,omitempty"`
	TokenKind string   `json:"token_kind"`
	SessionID string   `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

type Verifier struct {
	fetcher      Fetcher
	issuer       string
	audience     string
	now          func() time.Time
	refreshAfter time.Duration
	maxStale     time.Duration
	mu           sync.Mutex
	keys         map[string]ed25519.PublicKey
	loadedAt     time.Time
	cacheAge     time.Duration
	refreshing   bool
	refreshDone  chan struct{}
}

func NewVerifier(fetcher Fetcher, issuer, audience string) (*Verifier, error) {
	if fetcher == nil || strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" {
		return nil, errors.New("invalid JWKS verifier configuration")
	}
	return &Verifier{fetcher: fetcher, issuer: issuer, audience: audience, now: time.Now, refreshAfter: 5 * time.Minute, maxStale: 15 * time.Minute}, nil
}

func (v *Verifier) Check(ctx context.Context) error { return v.refresh(ctx) }

func (v *Verifier) Verify(ctx context.Context, raw string) (domain.Actor, error) {
	if strings.TrimSpace(raw) == "" {
		return domain.Actor{}, errors.New("empty access token")
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}), jwt.WithAudience(v.audience), jwt.WithIssuer(v.issuer), jwt.WithLeeway(30*time.Second))
	claims := &Claims{}
	token, err := parser.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || strings.TrimSpace(kid) == "" {
			return nil, errors.New("missing key id")
		}
		key, found := v.key(ctx, kid)
		if !found {
			return nil, errors.New("unknown key id")
		}
		return key, nil
	})
	if err != nil || token == nil || !token.Valid || claims.TokenKind != "human" || claims.Subject == "" || claims.ID == "" {
		return domain.Actor{}, errors.New("invalid access token")
	}
	return domain.Actor{Subject: claims.Subject, Type: "human", Roles: claims.Roles, Scopes: claims.Scopes, SessionID: claims.SessionID}, nil
}

func (v *Verifier) key(ctx context.Context, kid string) (ed25519.PublicKey, bool) {
	v.mu.Lock()
	key, found := v.keys[kid]
	stale := found && !v.usableLocked()
	v.mu.Unlock()
	if found && !stale {
		return key, true
	}
	if err := v.refresh(ctx); err != nil {
		return nil, false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	key, found = v.keys[kid]
	return key, found && v.usableLocked()
}

func (v *Verifier) refresh(ctx context.Context) error {
	if v.isFresh() {
		return nil
	}
	v.mu.Lock()
	if v.isFreshLocked() {
		v.mu.Unlock()
		return nil
	}
	if v.refreshing {
		done := v.refreshDone
		v.mu.Unlock()
		select {
		case <-done:
			return v.refreshError()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	v.refreshing = true
	v.refreshDone = make(chan struct{})
	done := v.refreshDone
	v.mu.Unlock()
	keys, age, err := v.fetcher.Fetch(ctx)
	v.mu.Lock()
	if err == nil {
		v.keys, v.loadedAt, v.cacheAge = keys, v.now().UTC(), age
	}
	v.refreshing = false
	close(done)
	v.mu.Unlock()
	return err
}

// refreshError intentionally reports readiness from the cache after a concurrent refresh.
func (v *Verifier) refreshError() error {
	if v.isFresh() {
		return nil
	}
	return ErrNotReady
}
func (v *Verifier) isFresh() bool { v.mu.Lock(); defer v.mu.Unlock(); return v.isFreshLocked() }
func (v *Verifier) isFreshLocked() bool {
	return len(v.keys) > 0 && v.now().UTC().Sub(v.loadedAt) <= v.effectiveRefreshAfterLocked()
}
func (v *Verifier) usableLocked() bool {
	return len(v.keys) > 0 && v.now().UTC().Sub(v.loadedAt) <= v.maxStale
}
func (v *Verifier) effectiveRefreshAfterLocked() time.Duration {
	if v.cacheAge > 0 && v.cacheAge < v.refreshAfter {
		return v.cacheAge
	}
	return v.refreshAfter
}
