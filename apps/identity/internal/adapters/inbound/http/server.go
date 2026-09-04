package httpadapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	identitycrypto "github.com/chouaib-skitou/streamweave/apps/identity/internal/adapters/outbound/crypto"
	"github.com/chouaib-skitou/streamweave/apps/identity/internal/application/health"
	identityapp "github.com/chouaib-skitou/streamweave/apps/identity/internal/application/identity"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Server struct {
	httpServer          *http.Server
	health              *health.Service
	logger              *slog.Logger
	requests            *prometheus.CounterVec
	authentication      *prometheus.CounterVec
	refreshes           *prometheus.CounterVec
	refreshReuse        prometheus.Counter
	jwtVerification     *prometheus.CounterVec
	sessionRevoked      *prometheus.CounterVec
	outboxPublished     *prometheus.CounterVec
	rateLimited         *prometheus.CounterVec
	dbPoolInUse         prometheus.Gauge
	dbPoolWaitCount     prometheus.Gauge
	outboxBacklog       prometheus.Gauge
	identity            *identityapp.Service
	signer              *identitycrypto.Signer
	revocations         RevocationChecker
	mux                 *http.ServeMux
	demoRegistration    bool
	humanAudience       string
	machineAudience     string
	gatewaySubject      string
	mailbox             identityapp.Mailbox
	testMailer          bool
	trustProxyHeaders   bool
	emergencyRevocation bool
	dbStats             func() sql.DBStats
	openAPIPath         string
}

type RevocationChecker interface {
	IsRevoked(context.Context, string) (bool, error)
}

func NewApplicationServer(addr, metricsPath string, readinessTimeout time.Duration, healthService *health.Service, logger *slog.Logger, service *identityapp.Service, signer *identitycrypto.Signer, revocations RevocationChecker, humanAudience string, demoRegistration, testMailer bool, mailbox identityapp.Mailbox) *Server {
	server := NewServer(addr, metricsPath, readinessTimeout, healthService, logger)
	server.identity, server.signer, server.revocations, server.humanAudience, server.demoRegistration = service, signer, revocations, humanAudience, demoRegistration
	server.testMailer, server.mailbox = testMailer, mailbox
	server.registerIdentityRoutes()
	return server
}

func NewServer(addr, metricsPath string, readinessTimeout time.Duration, healthService *health.Service, logger *slog.Logger) *Server {
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "identity",
		Name:      "http_requests_total",
		Help:      "Total HTTP requests handled by Identity.",
	}, []string{"route", "method", "status"})
	registry := prometheus.NewRegistry()
	authentication := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "identity", Name: "authentication_total", Help: "Authentication outcomes."}, []string{"outcome"})
	refreshes := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "identity", Name: "refresh_total", Help: "Refresh outcomes."}, []string{"outcome"})
	refreshReuse := prometheus.NewCounter(prometheus.CounterOpts{Namespace: "identity", Name: "refresh_reuse_total", Help: "Refresh token reuse detections."})
	jwtVerification := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "identity", Name: "jwt_verification_total", Help: "JWT verification outcomes."}, []string{"outcome"})
	sessionRevoked := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "identity", Name: "sessions_revoked_total", Help: "Session revocations initiated through the API."}, []string{"reason"})
	outboxPublished := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "identity", Name: "outbox_publish_total", Help: "Transactional outbox publication outcomes."}, []string{"outcome"})
	rateLimited := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "identity", Name: "rate_limited_total", Help: "Rate-limited requests."}, []string{"route"})
	dbPoolInUse := prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "identity", Name: "db_pool_in_use", Help: "Connections currently in use by the database pool."})
	dbPoolWaitCount := prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "identity", Name: "db_pool_wait_count", Help: "Database pool wait count."})
	outboxBacklog := prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "identity", Name: "outbox_backlog", Help: "Unpublished transactional outbox events."})
	registry.MustRegister(requests, authentication, refreshes, refreshReuse, jwtVerification, sessionRevoked, outboxPublished, rateLimited, dbPoolInUse, dbPoolWaitCount, outboxBacklog)

	mux := http.NewServeMux()
	server := &Server{health: healthService, logger: logger, requests: requests, authentication: authentication, refreshes: refreshes, refreshReuse: refreshReuse, jwtVerification: jwtVerification, sessionRevoked: sessionRevoked, outboxPublished: outboxPublished, rateLimited: rateLimited, dbPoolInUse: dbPoolInUse, dbPoolWaitCount: dbPoolWaitCount, outboxBacklog: outboxBacklog, mux: mux, openAPIPath: "contracts/openapi/identity.openapi.yaml"}
	mux.HandleFunc("GET /health/live", server.live)
	mux.HandleFunc("GET /health/startup", server.startup)
	mux.HandleFunc("GET /health/ready", server.ready(readinessTimeout))
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("GET /v1/docs", server.swaggerDocs)
	mux.HandleFunc("GET /v1/openapi.yaml", server.openAPI)

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

func (s *Server) SetDBStatsProvider(provider func() sql.DBStats) { s.dbStats = provider }
func (s *Server) SetOpenAPIPath(path string) {
	if path != "" {
		s.openAPIPath = path
	}
}
func (s *Server) SetOutboxBacklog(value int64) { s.outboxBacklog.Set(float64(value)) }
func (s *Server) ObserveSessionRevocation(reason string) {
	s.sessionRevoked.WithLabelValues(reason).Inc()
}
func (s *Server) ObserveOutboxPublish(outcome string) {
	s.outboxPublished.WithLabelValues(outcome).Inc()
}

func (s *Server) SetTrustProxyHeaders(enabled bool)       { s.trustProxyHeaders = enabled }
func (s *Server) SetEmergencyRevocation(enabled bool)     { s.emergencyRevocation = enabled }
func (s *Server) SetMachineAudience(audience string)      { s.machineAudience = audience }
func (s *Server) SetGatewayServiceSubject(subject string) { s.gatewaySubject = subject }

func (s *Server) requestSource(request *http.Request) string {
	return requestSource(request, s.trustProxyHeaders)
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
		correlationID := request.Header.Get("X-Correlation-ID")
		if correlationID == "" || len(correlationID) > 128 {
			correlationID = uuid.NewString()
		}
		writer.Header().Set("X-Correlation-ID", correlationID)
		request = request.WithContext(context.WithValue(request.Context(), correlationIDContextKey{}, correlationID))
		request = request.WithContext(otel.GetTextMapPropagator().Extract(request.Context(), propagation.HeaderCarrier(request.Header)))
		ctx, span := otel.Tracer("identity/http").Start(request.Context(), "HTTP "+request.Method, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attribute.String("http.request.method", request.Method)))
		defer span.End()
		request = request.WithContext(ctx)
		started := time.Now()
		wrapped := &responseWriter{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(wrapped, request)
		route := request.Pattern
		if route == "" {
			route = "unmatched"
		}
		span.SetName(request.Method + " " + route)
		span.SetAttributes(attribute.String("http.route", route))
		status := http.StatusText(wrapped.status)
		s.requests.WithLabelValues(route, request.Method, status).Inc()
		if wrapped.status == http.StatusTooManyRequests {
			s.rateLimited.WithLabelValues(route).Inc()
		}
		span.SetAttributes(attribute.Int("http.response.status_code", wrapped.status))
		if wrapped.status >= 500 {
			span.SetStatus(codes.Error, status)
		}
		if s.dbStats != nil {
			stats := s.dbStats()
			s.dbPoolInUse.Set(float64(stats.InUse))
			s.dbPoolWaitCount.Set(float64(stats.WaitCount))
		}
		spanContext := trace.SpanContextFromContext(request.Context())
		actorID := ""
		if claims := requestClaims(request); claims != nil {
			actorID = claims.Subject
		}
		s.logger.Info("http request", "method", request.Method, "route", route, "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds(), "correlation_id", correlationID, "trace_id", spanContext.TraceID().String(), "span_id", spanContext.SpanID().String(), "actor_id", actorID)
	})
}

type correlationIDContextKey struct{}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
