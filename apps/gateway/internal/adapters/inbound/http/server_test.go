package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	app "github.com/chouaib-skitou/streamweave/apps/gateway/internal/application/gateway"
	domain "github.com/chouaib-skitou/streamweave/apps/gateway/internal/domain/gateway"
)

type verifierStub struct {
	actor    domain.Actor
	err      error
	checkErr error
}

func (v verifierStub) Verify(context.Context, string) (domain.Actor, error) { return v.actor, v.err }
func (v verifierStub) Check(context.Context) error                          { return v.checkErr }

type limiterStub struct {
	allowed  bool
	err      error
	checkErr error
}

func (l limiterStub) Allow(context.Context, string, string) (bool, error) { return l.allowed, l.err }
func (l limiterStub) Check(context.Context) error                         { return l.checkErr }

type tokenStub struct{ err error }

func (t tokenStub) Token(context.Context) (string, error) {
	if t.err != nil {
		return "", t.err
	}
	return "svc", nil
}

type upstreamStub struct {
	response *domain.Response
	err      error
	input    domain.ForwardRequest
}

func (u *upstreamStub) Forward(_ context.Context, _ string, _ string, _ *domain.Actor, input domain.ForwardRequest) (*domain.Response, error) {
	u.input = input
	return u.response, u.err
}

func newTestServer(t *testing.T, verifier verifierStub, limiter limiterStub, upstream *upstreamStub, demo bool) *Server {
	t.Helper()
	service, err := app.NewService(verifier, limiter, tokenStub{}, upstream)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "gateway.openapi.yaml")
	if err := os.WriteFile(path, []byte("openapi: 3.1.0"), 0600); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer("127.0.0.1:0", "/metrics", time.Second, service, verifier, limiter, nil, path, demo, false, []string{"https://app.example"})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func request(server *Server, method, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, req)
	return response
}

func TestRoutesHealthDocsAndUnknowns(t *testing.T) {
	server := newTestServer(t, verifierStub{}, limiterStub{allowed: true}, &upstreamStub{response: &domain.Response{StatusCode: 200, Body: []byte(`{"ok":true}`)}}, false)
	if response := request(server, http.MethodGet, "/health/live", ""); response.Code != 200 {
		t.Fatalf("live=%d", response.Code)
	}
	if response := request(server, http.MethodGet, "/health/ready", ""); response.Code != 200 {
		t.Fatalf("ready=%d", response.Code)
	}
	if response := request(server, http.MethodGet, "/v1/docs", ""); response.Code != 200 {
		t.Fatalf("docs=%d", response.Code)
	}
	if response := request(server, http.MethodGet, "/v1/openapi.yaml", ""); response.Code != 200 {
		t.Fatalf("openapi=%d", response.Code)
	}
	if response := request(server, http.MethodGet, "/nope", ""); response.Code != 404 {
		t.Fatalf("not found=%d", response.Code)
	}
	if response := request(server, http.MethodGet, "/v1/auth/login", ""); response.Code != 405 {
		t.Fatalf("method=%d", response.Code)
	}
	if response := request(server, http.MethodOptions, "/v1/auth/login", ""); response.Code != 405 {
		t.Fatalf("options=%d", response.Code)
	}
}

func TestProtectedRouteAndOrderValidation(t *testing.T) {
	upstream := &upstreamStub{response: &domain.Response{StatusCode: 202, Header: map[string][]string{"Content-Type": {"application/json"}}, Body: []byte(`{"accepted":true}`)}}
	server := newTestServer(t, verifierStub{actor: domain.Actor{Subject: "user-1", Type: "human", Scopes: []string{"orders:write:self"}}}, limiterStub{allowed: true}, upstream, false)
	missing := request(server, http.MethodPost, "/v1/orders", `{"customer_id":"user-1"}`)
	if missing.Code != 400 {
		t.Fatalf("missing idempotency=%d", missing.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(`{"customer_id":"user-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Idempotency-Key", "key-1")
	req.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, req)
	if response.Code != 202 || upstream.input.IdempotencyKey != "key-1" || string(upstream.input.Body) == "" {
		t.Fatalf("forward code=%d request=%+v", response.Code, upstream.input)
	}
	badMedia := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader("x"))
	badMedia.Header.Set("Authorization", "Bearer token")
	badMedia.Header.Set("Idempotency-Key", "key")
	badMedia.RemoteAddr = "192.0.2.10:1234"
	response = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, badMedia)
	if response.Code != 415 {
		t.Fatalf("media=%d", response.Code)
	}
}

func TestFailuresAndCORS(t *testing.T) {
	server := newTestServer(t, verifierStub{}, limiterStub{allowed: true}, &upstreamStub{response: &domain.Response{StatusCode: 500, Header: map[string][]string{"Content-Type": {"text/plain"}}}}, false)
	unauth := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	unauth.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, unauth)
	if response.Code != 401 {
		t.Fatalf("unauth=%d", response.Code)
	}
	readyServer := newTestServer(t, verifierStub{checkErr: errors.New("jwks")}, limiterStub{allowed: true}, &upstreamStub{}, false)
	if response := request(readyServer, http.MethodGet, "/health/ready", ""); response.Code != 503 {
		t.Fatalf("not ready=%d", response.Code)
	}
	rateServer := newTestServer(t, verifierStub{}, limiterStub{allowed: false}, &upstreamStub{}, false)
	if response := request(rateServer, http.MethodPost, "/v1/auth/login", "{}"); response.Code != 429 {
		t.Fatalf("rate limit=%d", response.Code)
	}
	options := httptest.NewRequest(http.MethodOptions, "/v1/auth/login", nil)
	options.Header.Set("Origin", "https://app.example")
	response = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, options)
	if response.Code != 204 {
		t.Fatalf("cors=%d", response.Code)
	}
}
