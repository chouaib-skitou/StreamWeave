package gateway

import (
	"net/http"
	"strings"
	"time"
)

type AuthMode string

const (
	AuthNone   AuthMode = "none"
	AuthBearer AuthMode = "bearer"
)

type RoutePolicy struct {
	Method        string
	Pattern       string
	Owner         string
	Auth          AuthMode
	Scopes        []string
	LimitClass    string
	Timeout       time.Duration
	RetrySafeRead bool
	Mutation      bool
	MaxBodyBytes  int64
}

func (p RoutePolicy) Allows(scopes []string) bool {
	if len(p.Scopes) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		set[scope] = struct{}{}
	}
	for _, required := range p.Scopes {
		if _, ok := set[required]; ok {
			return true
		}
	}
	return false
}

func (p RoutePolicy) IsMutation() bool {
	return p.Mutation || p.Method == http.MethodPost || p.Method == http.MethodPut || p.Method == http.MethodPatch || p.Method == http.MethodDelete
}

func (p RoutePolicy) LimitKeyClass() string {
	if strings.TrimSpace(p.LimitClass) == "" {
		return "default"
	}
	return p.LimitClass
}

func DefaultPolicies() []RoutePolicy {
	return []RoutePolicy{
		{Method: http.MethodGet, Pattern: "/health/live", Owner: "gateway", Auth: AuthNone, LimitClass: "health", Timeout: time.Second, MaxBodyBytes: 0},
		{Method: http.MethodGet, Pattern: "/health/ready", Owner: "gateway", Auth: AuthNone, LimitClass: "health", Timeout: 2 * time.Second, MaxBodyBytes: 0},
		{Method: http.MethodGet, Pattern: "/metrics", Owner: "gateway", Auth: AuthNone, LimitClass: "metrics", Timeout: 2 * time.Second, MaxBodyBytes: 0},
		{Method: http.MethodGet, Pattern: "/v1/docs", Owner: "gateway", Auth: AuthNone, LimitClass: "health", Timeout: 2 * time.Second, MaxBodyBytes: 0},
		{Method: http.MethodGet, Pattern: "/v1/openapi.yaml", Owner: "gateway", Auth: AuthNone, LimitClass: "health", Timeout: 2 * time.Second, MaxBodyBytes: 0},
		{Method: http.MethodPost, Pattern: "/.well-known/register", Owner: "identity", Auth: AuthNone, LimitClass: "demo-registration", Timeout: 5 * time.Second, MaxBodyBytes: 1 << 20},
		{Method: http.MethodPost, Pattern: "/v1/auth/login", Owner: "identity", Auth: AuthNone, LimitClass: "auth-write", Timeout: 5 * time.Second, Mutation: true, MaxBodyBytes: 1 << 20},
		{Method: http.MethodPost, Pattern: "/v1/auth/refresh", Owner: "identity", Auth: AuthNone, LimitClass: "auth-write", Timeout: 5 * time.Second, Mutation: true, MaxBodyBytes: 1 << 20},
		{Method: http.MethodPost, Pattern: "/v1/auth/password-reset/request", Owner: "identity", Auth: AuthNone, LimitClass: "auth-write", Timeout: 5 * time.Second, Mutation: true, MaxBodyBytes: 1 << 20},
		{Method: http.MethodPost, Pattern: "/v1/auth/password-reset/confirm", Owner: "identity", Auth: AuthNone, LimitClass: "auth-write", Timeout: 5 * time.Second, Mutation: true, MaxBodyBytes: 1 << 20},
		{Method: http.MethodPost, Pattern: "/v1/auth/email-verification/confirm", Owner: "identity", Auth: AuthNone, LimitClass: "auth-write", Timeout: 5 * time.Second, Mutation: true, MaxBodyBytes: 1 << 20},
		{Method: http.MethodPost, Pattern: "/v1/auth/logout", Owner: "identity", Auth: AuthBearer, Scopes: []string{"identity:sessions:write"}, LimitClass: "auth-write", Timeout: 5 * time.Second, Mutation: true, MaxBodyBytes: 1 << 20},
		{Method: http.MethodPost, Pattern: "/v1/auth/logout-all", Owner: "identity", Auth: AuthBearer, Scopes: []string{"identity:sessions:write"}, LimitClass: "auth-write", Timeout: 5 * time.Second, Mutation: true, MaxBodyBytes: 1 << 20},
		{Method: http.MethodGet, Pattern: "/v1/auth/sessions", Owner: "identity", Auth: AuthBearer, Scopes: []string{"identity:sessions:read"}, LimitClass: "authenticated-read", Timeout: 5 * time.Second, RetrySafeRead: true, MaxBodyBytes: 0},
		{Method: http.MethodPost, Pattern: "/v1/orders", Owner: "orders", Auth: AuthBearer, Scopes: []string{"orders:write:self", "orders:write:any"}, LimitClass: "order-write", Timeout: 10 * time.Second, Mutation: true, MaxBodyBytes: 2 << 20},
		{Method: http.MethodGet, Pattern: "/v1/orders", Owner: "orders", Auth: AuthBearer, Scopes: []string{"orders:read:self", "orders:read:any"}, LimitClass: "order-read", Timeout: 5 * time.Second, RetrySafeRead: true, MaxBodyBytes: 0},
		{Method: http.MethodGet, Pattern: "/v1/orders/{order_id}", Owner: "orders", Auth: AuthBearer, Scopes: []string{"orders:read:self", "orders:read:any"}, LimitClass: "order-read", Timeout: 5 * time.Second, RetrySafeRead: true, MaxBodyBytes: 0},
		{Method: http.MethodPost, Pattern: "/v1/orders/{order_id}/cancel", Owner: "orders", Auth: AuthBearer, Scopes: []string{"orders:cancel:self", "orders:cancel"}, LimitClass: "order-write", Timeout: 10 * time.Second, Mutation: true, MaxBodyBytes: 1 << 20},
	}
}
