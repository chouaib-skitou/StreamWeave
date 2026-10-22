package postgres

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	app "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/orders"
)

type quoteItem struct {
	ProductID      string `json:"product_id"`
	Quantity       int32  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	LineTotalMinor int64  `json:"line_total_minor"`
}

type signedQuote struct {
	OrderID        string      `json:"order_id"`
	ReservationID  string      `json:"reservation_id"`
	QuoteID        string      `json:"quote_id"`
	QuoteVersion   int64       `json:"quote_version"`
	Currency       string      `json:"currency"`
	Items          []quoteItem `json:"items"`
	QuoteExpiresAt string      `json:"quote_expires_at"`
}

// QuoteVerifier validates Inventory's detached EdDSA quote signature and expiry.
type QuoteVerifier struct {
	url    string
	client *http.Client
	mu     sync.Mutex
	keys   map[string]ed25519.PublicKey
	loaded time.Time
}

func NewQuoteVerifier(jwksURL string) *QuoteVerifier {
	return &QuoteVerifier{url: jwksURL, client: &http.Client{Timeout: 2 * time.Second}}
}

func (v *QuoteVerifier) Verify(ctx context.Context, quote signedQuote, keyID, detachedJWS string, now time.Time) error {
	if v == nil || strings.TrimSpace(v.url) == "" || keyID == "" {
		return fmt.Errorf("%w: quote verification is not configured", app.ErrConflict)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, quote.QuoteExpiresAt)
	if err != nil || !expiresAt.After(now.UTC()) {
		return fmt.Errorf("%w: quote expired or invalid", app.ErrConflict)
	}
	parts := strings.Split(detachedJWS, ".")
	if len(parts) != 3 || parts[1] != "" {
		return fmt.Errorf("%w: quote signature is not detached JWS", app.ErrConflict)
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "EdDSA" || header.KeyID != keyID {
		return fmt.Errorf("%w: quote signature header is invalid", app.ErrConflict)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("%w: quote signature is invalid", app.ErrConflict)
	}
	canonical, err := json.Marshal(quote)
	if err != nil {
		return err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(canonical)
	publicKey, ok := v.key(ctx, keyID)
	if !ok || !ed25519.Verify(publicKey, []byte(parts[0]+"."+encodedPayload), signature) {
		return fmt.Errorf("%w: quote signature verification failed", app.ErrConflict)
	}
	return nil
}

func (v *QuoteVerifier) key(ctx context.Context, keyID string) (ed25519.PublicKey, bool) {
	v.mu.Lock()
	if key, ok := v.keys[keyID]; ok && time.Since(v.loaded) < 5*time.Minute {
		v.mu.Unlock()
		return key, true
	}
	v.mu.Unlock()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.url, nil)
	if err != nil {
		return nil, false
	}
	response, err := v.client.Do(request)
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
		Keys []struct {
			KTY string `json:"kty"`
			CRV string `json:"crv"`
			KID string `json:"kid"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			X   string `json:"x"`
		} `json:"keys"`
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
	v.mu.Lock()
	v.keys = keys
	v.loaded = time.Now()
	key, ok := keys[keyID]
	v.mu.Unlock()
	return key, ok
}
