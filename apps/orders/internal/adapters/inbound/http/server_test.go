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

	apphealth "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/health"
)

type dependency struct{ err error }

func (d dependency) Name() string                { return "database" }
func (d dependency) Check(context.Context) error { return d.err }

func newServer(t *testing.T, deps ...dependency) *Server {
	t.Helper()
	path := filepath.Join(t.TempDir(), "orders.openapi.yaml")
	if err := os.WriteFile(path, []byte("openapi: 3.1.0"), 0600); err != nil {
		t.Fatal(err)
	}
	checks := make([]apphealth.Dependency, len(deps))
	for i := range deps {
		checks[i] = deps[i]
	}
	s, err := NewServer("127.0.0.1:0", "/metrics", time.Second, apphealth.NewService(checks...), nil, path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func request(s *Server, method, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, r)
	return w
}

func TestHealthDocsMetricsAndUnknownRoutes(t *testing.T) {
	s := newServer(t)
	for _, path := range []string{"/health/live", "/health/ready", "/v1/docs", "/v1/openapi.yaml", "/metrics"} {
		if got := request(s, http.MethodGet, path).Code; got != http.StatusOK {
			t.Fatalf("%s=%d", path, got)
		}
	}
	if got := request(s, http.MethodGet, "/missing").Code; got != http.StatusNotFound {
		t.Fatalf("missing=%d", got)
	}
	if got := request(s, http.MethodPost, "/health/live").Code; got != http.StatusMethodNotAllowed {
		t.Fatalf("method=%d", got)
	}
}

func TestReadinessFailureAndOpenAPIFailure(t *testing.T) {
	s := newServer(t, dependency{err: errors.New("database unavailable")})
	if got := request(s, http.MethodGet, "/health/ready").Code; got != http.StatusServiceUnavailable {
		t.Fatalf("ready=%d", got)
	}
	s.openAPIPath = filepath.Join(t.TempDir(), "missing.yaml")
	if got := request(s, http.MethodGet, "/v1/openapi.yaml").Code; got != http.StatusServiceUnavailable {
		t.Fatalf("openapi=%d", got)
	}
}

func TestRequestIDIsGeneratedOrPreserved(t *testing.T) {
	s := newServer(t)
	r := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	r.Header.Set("X-Request-ID", strings.Repeat("x", 129))
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, r)
	if w.Header().Get("X-Request-ID") == strings.Repeat("x", 129) || w.Header().Get("X-Request-ID") == "" {
		t.Fatal("invalid request id was retained")
	}
	r = httptest.NewRequest(http.MethodGet, "/health/live", nil)
	r.Header.Set("X-Request-ID", "client-request")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, r)
	if w.Header().Get("X-Request-ID") != "client-request" {
		t.Fatal("valid request id was not retained")
	}
}
