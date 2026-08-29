package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	identitycrypto "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/crypto"
	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/application/health"
	identityapp "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/application/identity"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	httpServer       *http.Server
	health           *health.Service
	logger           *slog.Logger
	requests         *prometheus.CounterVec
	identity         *identityapp.Service
	signer           *identitycrypto.Signer
	revocations      RevocationChecker
	mux              *http.ServeMux
	demoRegistration bool
	humanAudience    string
}

type RevocationChecker interface {
	IsRevoked(context.Context, string) (bool, error)
}

func NewApplicationServer(addr, metricsPath string, readinessTimeout time.Duration, healthService *health.Service, logger *slog.Logger, service *identityapp.Service, signer *identitycrypto.Signer, revocations RevocationChecker, humanAudience string, demoRegistration bool) *Server {
	server := NewServer(addr, metricsPath, readinessTimeout, healthService, logger)
	server.identity, server.signer, server.revocations, server.humanAudience, server.demoRegistration = service, signer, revocations, humanAudience, demoRegistration
	server.registerIdentityRoutes()
	return server
}

func NewServer(addr, metricsPath string, readinessTimeout time.Duration, healthService *health.Service, logger *slog.Logger) *Server {
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "identity",
		Name:      "http_requests_total",
		Help:      "Total HTTP requests handled by Identity.",
	}, []string{"method", "status"})
	registry := prometheus.NewRegistry()
	registry.MustRegister(requests)

	mux := http.NewServeMux()
	server := &Server{health: healthService, logger: logger, requests: requests, mux: mux}
	mux.HandleFunc("GET /health/live", server.live)
	mux.HandleFunc("GET /health/startup", server.startup)
	mux.HandleFunc("GET /health/ready", server.ready(readinessTimeout))
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server.httpServer = &http.Server{
		Addr:              addr,
		Handler:           server.accessLog(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server
}

func (s *Server) Start() error {
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) live(writer http.ResponseWriter, request *http.Request) {
	s.writeHealth(writer, http.StatusOK, s.health.Live())
}

func (s *Server) startup(writer http.ResponseWriter, request *http.Request) {
	result := s.health.Startup()
	status := http.StatusOK
	if result.Status != "ok" {
		status = http.StatusServiceUnavailable
	}
	s.writeHealth(writer, status, result)
}

func (s *Server) ready(timeout time.Duration) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), timeout)
		defer cancel()
		result := s.health.Ready(ctx)
		status := http.StatusOK
		if result.Status != "ready" {
			status = http.StatusServiceUnavailable
		}
		s.writeHealth(writer, status, result)
	}
}

func (s *Server) writeHealth(writer http.ResponseWriter, status int, value health.Result) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		wrapped := &responseWriter{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(wrapped, request)
		s.requests.WithLabelValues(request.Method, http.StatusText(wrapped.status)).Inc()
		s.logger.Info("http request", "method", request.Method, "route", request.URL.Path, "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
