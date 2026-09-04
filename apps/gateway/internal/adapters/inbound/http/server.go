package httpadapter

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	app "github.com/chouaib-skitou/streamweave/apps/gateway/internal/application/gateway"
	domain "github.com/chouaib-skitou/streamweave/apps/gateway/internal/domain/gateway"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type HealthChecker interface{ Check(context.Context) error }

type Server struct {
	httpServer        *http.Server
	service           *app.Service
	verifier          HealthChecker
	limiter           HealthChecker
	logger            *slog.Logger
	policies          []domain.RoutePolicy
	byPattern         map[string]domain.RoutePolicy
	openAPIPath       string
	demoRegistration  bool
	trustProxyHeaders bool
	corsOrigins       map[string]struct{}
	requests          *prometheus.CounterVec
	upstreamFailures  *prometheus.CounterVec
	authFailures      *prometheus.CounterVec
	rateLimited       *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	inFlight          prometheus.Gauge
	metrics           *gatewayMetrics
	readinessTimeout  time.Duration
	metricsPath       string
}

type gatewayMetrics struct {
	authFailures     *prometheus.CounterVec
	rateLimited      *prometheus.CounterVec
	limiterErrors    *prometheus.CounterVec
	jwksRefresh      *prometheus.CounterVec
	upstreamRequests *prometheus.CounterVec
	upstreamDuration *prometheus.HistogramVec
}

func (m *gatewayMetrics) ObserveAuthFailure(route, reason string) {
	m.authFailures.WithLabelValues(route, reason).Inc()
}
func (m *gatewayMetrics) ObserveRateLimitRejection(route, class string) {
	m.rateLimited.WithLabelValues(route, class).Inc()
}
func (m *gatewayMetrics) ObserveRateLimiterError(class string) {
	m.limiterErrors.WithLabelValues(class).Inc()
}
func (m *gatewayMetrics) ObserveJWKSRefresh(outcome string) {
	m.jwksRefresh.WithLabelValues(outcome).Inc()
}
func (m *gatewayMetrics) ObserveUpstreamRequest(service, route, outcome string) {
	m.upstreamRequests.WithLabelValues(service, route, outcome).Inc()
}
func (m *gatewayMetrics) ObserveUpstreamDuration(service, route string, duration time.Duration) {
	m.upstreamDuration.WithLabelValues(service, route).Observe(duration.Seconds())
}

func NewServer(addr, metricsPath string, readinessTimeout time.Duration, service *app.Service, verifier, limiter HealthChecker, logger *slog.Logger, openAPIPath string, demoRegistration, trustProxyHeaders bool, corsOrigins []string) (*Server, error) {
	if service == nil || verifier == nil || limiter == nil {
		return nil, errors.New("gateway HTTP dependencies are required")
	}
	if metricsPath != "/metrics" {
		return nil, errors.New("Gateway metrics path must be /metrics")
	}
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "gateway", Name: "http_requests_total", Help: "Total HTTP requests handled by Gateway."}, []string{"route", "method", "status"})
	upstreamFailures := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "gateway", Name: "upstream_failures_total", Help: "Upstream failures by safe category."}, []string{"owner", "reason"})
	authFailures := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "gateway", Name: "auth_failures_total", Help: "Gateway authorization failures."}, []string{"route", "reason_class"})
	rateLimited := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "gateway", Name: "rate_limit_rejections_total", Help: "Gateway rate-limited requests."}, []string{"route", "limit_class"})
	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "gateway", Name: "http_request_duration_seconds", Help: "Gateway HTTP request duration."}, []string{"route", "method"})
	inFlight := prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "gateway", Name: "in_flight_requests", Help: "Gateway requests currently being handled."})
	limiterErrors := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "gateway", Name: "rate_limiter_errors_total", Help: "Gateway rate limiter errors."}, []string{"limit_class"})
	jwksRefresh := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "gateway", Name: "jwks_refresh_total", Help: "Gateway JWKS refresh outcomes."}, []string{"outcome"})
	upstreamRequests := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "gateway", Name: "upstream_requests_total", Help: "Gateway upstream request outcomes."}, []string{"service", "route", "outcome"})
	upstreamDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "gateway", Name: "upstream_duration_seconds", Help: "Gateway upstream request duration."}, []string{"service", "route"})
	registry.MustRegister(requests, upstreamFailures, authFailures, rateLimited, requestDuration, inFlight, limiterErrors, jwksRefresh, upstreamRequests, upstreamDuration)
	metrics := &gatewayMetrics{authFailures: authFailures, rateLimited: rateLimited, limiterErrors: limiterErrors, jwksRefresh: jwksRefresh, upstreamRequests: upstreamRequests, upstreamDuration: upstreamDuration}
	server := &Server{service: service, verifier: verifier, limiter: limiter, logger: logger, policies: domain.DefaultPolicies(), byPattern: make(map[string]domain.RoutePolicy), openAPIPath: openAPIPath, demoRegistration: demoRegistration, trustProxyHeaders: trustProxyHeaders, corsOrigins: make(map[string]struct{}), requests: requests, upstreamFailures: upstreamFailures, authFailures: authFailures, rateLimited: rateLimited, requestDuration: requestDuration, inFlight: inFlight, metrics: metrics, readinessTimeout: readinessTimeout, metricsPath: metricsPath}
	service.SetMetrics(metrics)
	if observer, ok := verifier.(interface{ SetRefreshObserver(func(string)) }); ok {
		observer.SetRefreshObserver(metrics.ObserveJWKSRefresh)
	}
	for _, policy := range server.policies {
		server.byPattern[policy.Pattern] = policy
	}
	for _, origin := range corsOrigins {
		if strings.TrimSpace(origin) != "" {
			server.corsOrigins[strings.TrimSpace(origin)] = struct{}{}
		}
	}
	mux := http.NewServeMux()
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	mux.Handle(metricsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		metricsHandler.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/", server.route)
	server.httpServer = &http.Server{Addr: addr, Handler: server.observability(server.cors(mux)), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	return server, nil
}

func (s *Server) Start() error {
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func (s *Server) Shutdown(ctx context.Context) error { return s.httpServer.Shutdown(ctx) }

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health/live" {
		if r.Method != http.MethodGet {
			problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		s.health(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if r.URL.Path == "/health/ready" {
		if r.Method != http.MethodGet {
			problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		s.ready(w, r)
		return
	}
	if r.URL.Path == "/v1/docs" {
		if r.Method != http.MethodGet {
			problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		s.docs(w, r)
		return
	}
	if r.URL.Path == "/v1/openapi.yaml" {
		if r.Method != http.MethodGet {
			problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		s.openapi(w, r)
		return
	}
	policy, ok := s.matchPolicy(r.Method, r.URL.Path)
	if !ok {
		if _, exists := s.matchPolicy("", r.URL.Path); exists {
			problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		problem(w, r, http.StatusNotFound, "not_found", "Route not found")
		return
	}
	if policy.Pattern == "/.well-known/register" && !s.demoRegistration {
		problem(w, r, http.StatusNotFound, "not_found", "Route not found")
		return
	}
	if holder, ok := r.Context().Value(routeHolderKey{}).(*string); ok {
		*holder = policy.Pattern
	}
	if err := validateRequest(w, r, policy); err != nil {
		s.metrics.ObserveAuthFailure(policy.Pattern, problemReason(err))
		return
	}
	source := requestSource(r, s.trustProxyHeaders)
	actor, _, err := s.service.Authorize(r.Context(), r, policy, source)
	if err != nil {
		s.writeServiceError(w, r, policy.Pattern, err)
		return
	}
	if actor.Subject != "" {
		r = r.WithContext(domain.WithActor(r.Context(), actor))
	}
	body, readErr := readBody(r, policy.MaxBodyBytes)
	if readErr != nil {
		problem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large")
		return
	}
	request := domain.ForwardRequest{Method: r.Method, Path: r.URL.Path, Route: policy.Pattern, RawQuery: r.URL.RawQuery, Body: body, ContentType: r.Header.Get("Content-Type"), RequestID: requestID(r), CorrelationID: correlationID(r), IdempotencyKey: r.Header.Get("Idempotency-Key")}
	forwardContext, cancel := context.WithTimeout(r.Context(), policy.Timeout)
	defer cancel()
	response, err := s.service.Forward(forwardContext, policy.Owner, actorPointer(actor), request, policy)
	if err != nil {
		s.writeServiceError(w, r, policy.Pattern, err)
		return
	}
	s.writeResponse(w, r, response)
}

func (s *Server) matchPolicy(method, path string) (domain.RoutePolicy, bool) {
	if method != "" {
		for _, policy := range s.policies {
			if policy.Method == method && routeMatches(policy.Pattern, path) {
				return policy, true
			}
		}
		return domain.RoutePolicy{}, false
	}
	for _, policy := range s.policies {
		if routeMatches(policy.Pattern, path) {
			return policy, true
		}
	}
	return domain.RoutePolicy{}, false
}

var orderIDPath = regexp.MustCompile(`^/v1/orders/ord_[A-Za-z0-9]+(?:/cancel)?$`)
var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{16,256}$`)
var customerIDPattern = regexp.MustCompile(`^cus_[A-Za-z0-9]+$`)

func routeMatches(pattern, path string) bool {
	if pattern == path {
		return true
	}
	if strings.HasSuffix(pattern, "/{order_id}") {
		return orderIDPath.MatchString(path) && !strings.HasSuffix(path, "/cancel")
	}
	if strings.HasSuffix(pattern, "/{order_id}/cancel") {
		return orderIDPath.MatchString(path) && strings.HasSuffix(path, "/cancel")
	}
	return false
}

func validateRequest(w http.ResponseWriter, r *http.Request, policy domain.RoutePolicy) error {
	if policy.MaxBodyBytes > 0 && r.ContentLength > policy.MaxBodyBytes {
		problem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large")
		return domain.ErrPayloadTooLarge
	}
	if policy.RequireJSON && !isJSONContentType(r.Header.Get("Content-Type")) {
		problem(w, r, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return domain.ErrUnsupportedMedia
	}
	if policy.Owner == "orders" && policy.IsMutation() {
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" {
			problem(w, r, http.StatusBadRequest, "bad_request", "Idempotency-Key is required")
			return domain.ErrBadRequest
		}
		if !idempotencyKeyPattern.MatchString(key) {
			problem(w, r, http.StatusBadRequest, "bad_request", "Idempotency-Key is invalid")
			return domain.ErrBadRequest
		}
	}
	if (policy.Pattern == "/v1/orders" || policy.Pattern == "/v1/auth/sessions") && r.Method == http.MethodGet {
		if err := validateOrderQuery(w, r); err != nil {
			return err
		}
	}
	return nil
}

func validateOrderQuery(w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			problem(w, r, http.StatusBadRequest, "bad_request", "limit must be between 1 and 100")
			return domain.ErrBadRequest
		}
	}
	if cursor := query.Get("cursor"); len(cursor) > 256 {
		problem(w, r, http.StatusBadRequest, "bad_request", "cursor is too long")
		return domain.ErrBadRequest
	}
	if customer := strings.TrimSpace(query.Get("customer_id")); customer != "" && !customerIDPattern.MatchString(customer) {
		problem(w, r, http.StatusBadRequest, "bad_request", "customer_id is invalid")
		return domain.ErrBadRequest
	}
	return nil
}

func readBody(r *http.Request, max int64) ([]byte, error) {
	if max <= 0 {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > max {
		return nil, domain.ErrPayloadTooLarge
	}
	return body, nil
}
func isJSONContentType(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "application/json" || strings.HasPrefix(value, "application/json;")
}
func actorPointer(actor domain.Actor) *domain.Actor {
	if actor.Subject == "" {
		return nil
	}
	return &actor
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.readinessTimeout)
	defer cancel()
	if err := s.verifier.Check(ctx); err != nil {
		problem(w, r, http.StatusServiceUnavailable, "dependency_unavailable", "JWKS dependency is unavailable")
		return
	}
	if err := s.limiter.Check(ctx); err != nil {
		problem(w, r, http.StatusServiceUnavailable, "dependency_unavailable", "Redis dependency is unavailable")
		return
	}
	s.health(w, http.StatusOK, map[string]string{"status": "ready"})
}
func (s *Server) health(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (s *Server) docs(w http.ResponseWriter, r *http.Request) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		problem(w, r, http.StatusInternalServerError, "internal_error", "Unable to initialize API documentation")
		return
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src https://unpkg.com 'unsafe-inline'; script-src https://unpkg.com 'nonce-"+nonce+"'; img-src https://unpkg.com data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>StreamWeave Gateway API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script nonce="`+nonce+`">window.onload=()=>window.ui=SwaggerUIBundle({url:'/v1/openapi.yaml',dom_id:'#swagger-ui',deepLinking:true,displayRequestDuration:true,persistAuthorization:false,filter:true,tryItOutEnabled:true});</script></body></html>`)
}
func (s *Server) openapi(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(s.openAPIPath)
	if err != nil {
		problem(w, r, http.StatusServiceUnavailable, "dependency_unavailable", "OpenAPI document unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) writeResponse(w http.ResponseWriter, r *http.Request, response *app.Response) {
	for name, values := range response.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	if response.StatusCode == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", boundedRetryAfter(w.Header().Get("Retry-After")))
	}
	if response.StatusCode >= 400 && !strings.Contains(strings.ToLower(w.Header().Get("Content-Type")), "application/problem+json") {
		problem(w, r, response.StatusCode, "upstream_error", "The requested operation was rejected")
		return
	}
	if response.StatusCode >= 400 {
		var upstream domain.Problem
		if err := json.Unmarshal(response.Body, &upstream); err != nil || len(upstream.Detail) > 512 || strings.ContainsAny(upstream.Detail, "\r\n") {
			problem(w, r, response.StatusCode, "upstream_error", "The requested operation was rejected")
			return
		}
		if !isSafeProblemType(upstream.Type) {
			upstream.Type = "https://streamweave.dev/problems/upstream_error"
		}
		if strings.TrimSpace(upstream.Detail) == "" || len(upstream.Detail) > 512 {
			upstream.Detail = "The requested operation was rejected"
		}
		if upstream.Title == "" || len(upstream.Title) > 128 || strings.ContainsAny(upstream.Title, "\r\n") {
			upstream.Title = http.StatusText(response.StatusCode)
		}
		upstream.Status = response.StatusCode
		upstream.Instance = r.URL.Path
		upstream.RequestID = requestID(r)
		writeProblem(w, upstream)
		return
	}
	w.WriteHeader(response.StatusCode)
	if r.Method != http.MethodHead && response.StatusCode != http.StatusNoContent {
		_, _ = w.Write(response.Body)
	}
}

func isSafeProblemType(value string) bool {
	if value == "about:blank" {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host == "streamweave.dev" && strings.HasPrefix(parsed.Path, "/problems/") && len(value) <= 256
}

func boundedRetryAfter(value string) string {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds < 1 || seconds > 60 {
		return "60"
	}
	return strconv.Itoa(seconds)
}

func (s *Server) writeServiceError(w http.ResponseWriter, r *http.Request, route string, err error) {
	status, reason, detail := mapError(err)
	if status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "60")
	}
	if status >= 500 {
		s.upstreamFailures.WithLabelValues(routeOwner(route), reason).Inc()
	}
	problem(w, r, status, reason, detail)
}
func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return 401, "unauthorized", "Authentication is required"
	case errors.Is(err, domain.ErrForbidden):
		return 403, "forbidden", "The requested operation is not permitted"
	case errors.Is(err, domain.ErrRateLimited):
		return 429, "rate_limited", "Too many requests"
	case errors.Is(err, domain.ErrDependency):
		return 503, "dependency_unavailable", "A required dependency is unavailable"
	case errors.Is(err, domain.ErrUpstreamTimeout):
		return 504, "upstream_timeout", "The upstream service did not respond in time"
	case errors.Is(err, domain.ErrUpstreamUnavailable):
		return 503, "upstream_unavailable", "The upstream service is unavailable"
	case errors.Is(err, domain.ErrUpstreamResponse):
		return 502, "upstream_error", "The upstream response was invalid"
	case errors.Is(err, domain.ErrPayloadTooLarge):
		return 413, "payload_too_large", "Request body is too large"
	case errors.Is(err, domain.ErrUnsupportedMedia):
		return 415, "unsupported_media_type", "Content-Type must be application/json"
	default:
		return 400, "bad_request", "The request is invalid"
	}
}
func problem(w http.ResponseWriter, r *http.Request, status int, reason, detail string) {
	value := domain.Problem{Type: "https://streamweave.dev/problems/" + reason, Title: http.StatusText(status), Status: status, Detail: detail, RequestID: requestID(r)}
	if r != nil {
		value.Instance = r.URL.Path
	}
	writeProblem(w, value)
}
func writeProblem(w http.ResponseWriter, value domain.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(value.Status)
	_ = json.NewEncoder(w).Encode(value)
}
func problemReason(err error) string { _, reason, _ := mapError(err); return reason }
func routeOwner(route string) string {
	if strings.HasPrefix(route, "/v1/orders") {
		return "orders"
	}
	return "identity"
}

type contextKey string

type routeHolderKey struct{}

const requestIDKey contextKey = "request-id"
const correlationIDKey contextKey = "correlation-id"

func requestID(r *http.Request) string {
	if r == nil {
		return ""
	}
	value, _ := r.Context().Value(requestIDKey).(string)
	return value
}
func correlationID(r *http.Request) string {
	value, _ := r.Context().Value(correlationIDKey).(string)
	return value
}

var acceptedID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

func validID(value string) bool { return acceptedID.MatchString(value) }
func makeRequestID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return "sw_" + uuid.NewString()
	}
	return "sw_" + id.String()
}
func requestSource(r *http.Request, trustProxy bool) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if trustProxy {
			if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); net.ParseIP(forwarded) != nil {
				return forwarded
			}
		}
		return host
	}
	return "unknown"
}
func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.inFlight.Inc()
		defer s.inFlight.Dec()
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if _, ok := s.corsOrigins[origin]; !ok {
				if r.Method == http.MethodOptions {
					problem(w, r, http.StatusForbidden, "forbidden", "Origin is not allowed")
					return
				}
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				if r.Method == http.MethodOptions && s.isKnownRoute(r.URL.Path) {
					w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Idempotency-Key,X-Request-ID,X-Correlation-ID")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) isKnownRoute(path string) bool {
	if path == "/metrics" || path == "/health/live" || path == "/health/ready" || path == "/v1/docs" || path == "/v1/openapi.yaml" {
		return true
	}
	_, ok := s.matchPolicy("", path)
	return ok
}

func (s *Server) observability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validID(id) {
			id = makeRequestID()
		}
		correlation := r.Header.Get("X-Correlation-ID")
		if !validID(correlation) {
			correlation = id
		}
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("X-Correlation-ID", correlation)
		routeHolder := new(string)
		ctx := context.WithValue(r.Context(), routeHolderKey{}, routeHolder)
		ctx = context.WithValue(ctx, requestIDKey, id)
		ctx = context.WithValue(ctx, correlationIDKey, correlation)
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
		ctx, span := otel.Tracer("gateway/http").Start(ctx, "HTTP "+r.Method, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attribute.String("http.request.method", r.Method)))
		defer span.End()
		started := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r.WithContext(ctx))
		route := *routeHolder
		if route == "" {
			route = r.URL.Path
		}
		span.SetName(r.Method + " " + route)
		span.SetAttributes(attribute.String("http.route", route), attribute.Int("http.response.status_code", wrapped.status))
		if wrapped.status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(wrapped.status))
		}
		spanContext := trace.SpanContextFromContext(ctx)
		s.requests.WithLabelValues(route, r.Method, strconv.Itoa(wrapped.status)).Inc()
		s.requestDuration.WithLabelValues(route, r.Method).Observe(time.Since(started).Seconds())
		s.logger.Info("http request", "method", r.Method, "route", route, "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds(), "request_id", id, "correlation_id", correlation, "trace_id", spanContext.TraceID().String(), "span_id", spanContext.SpanID().String(), "source", fingerprint(requestSource(r, s.trustProxyHeaders)))
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.status = status
	w.wrote = true
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseWriter) Write(body []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (s *Server) ObserveAuthFailure(route, reason string) {
	s.authFailures.WithLabelValues(route, reason).Inc()
}

func (s *Server) ObserveRateLimitRejection(route, class string) {
	s.rateLimited.WithLabelValues(route, class).Inc()
}

func (s *Server) ObserveRateLimiterError(class string) {
	s.upstreamFailures.WithLabelValues("gateway", "limiter_"+class).Inc()
}

func (s *Server) ObserveJWKSRefresh(string) {}

func (s *Server) ObserveUpstreamRequest(owner, route, outcome string) {
	if outcome != "success" {
		s.upstreamFailures.WithLabelValues(owner, outcome).Inc()
	}
}

func (s *Server) ObserveUpstreamDuration(string, string, time.Duration) {}
