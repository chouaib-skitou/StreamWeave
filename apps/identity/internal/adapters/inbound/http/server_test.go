package httpadapter

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/application/health"
)

type unavailableDependency struct{}

func (unavailableDependency) Name() string                { return "postgresql" }
func (unavailableDependency) Check(context.Context) error { return errors.New("unavailable") }

func TestHealthEndpointsExposeOperationalState(t *testing.T) {
	service := health.NewService()
	service.MarkStarted()
	server := NewServer(":0", "/metrics", time.Second, service, slog.Default())

	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/health/live", want: http.StatusOK},
		{path: "/health/startup", want: http.StatusOK},
		{path: "/health/ready", want: http.StatusOK},
		{path: "/metrics", want: http.StatusOK},
		{path: "/v1/docs", want: http.StatusOK},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		server.httpServer.Handler.ServeHTTP(recorder, request)
		if recorder.Code != test.want {
			t.Fatalf("%s: got %d, want %d", test.path, recorder.Code, test.want)
		}
	}
}

func TestSwaggerDocsHasSecurityHeaders(t *testing.T) {
	service := health.NewService()
	server := NewServer(":0", "/metrics", time.Second, service, slog.Default())
	recorder := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/docs", nil))
	if recorder.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("expected Content-Security-Policy header")
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("expected nosniff header")
	}
	if recorder.Body.String() == "" {
		t.Fatal("expected Swagger UI document")
	}
}

func TestReadinessReturnsServiceUnavailable(t *testing.T) {
	service := health.NewService(unavailableDependency{})
	server := NewServer(":0", "/metrics", time.Second, service, slog.Default())
	recorder := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
