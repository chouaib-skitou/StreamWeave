package httpadapter

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	apphealth "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/health"
	apporders "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/orders"
	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
)

type Authenticator interface {
	Authenticate(context.Context, http.Header) (apporders.Actor, error)
}

type StaticAuthenticator struct{ token string }

func NewStaticAuthenticator(token string) *StaticAuthenticator {
	return &StaticAuthenticator{token: token}
}

func (a *StaticAuthenticator) Authenticate(_ context.Context, headers http.Header) (apporders.Actor, error) {
	if a == nil || strings.TrimSpace(a.token) == "" {
		return apporders.Actor{}, apporders.ErrUnauthorized
	}
	authorization := headers.Values("Authorization")
	if len(authorization) != 1 || authorization[0] != "Bearer "+a.token {
		return apporders.Actor{}, apporders.ErrUnauthorized
	}
	subject := strings.TrimSpace(headers.Get("X-StreamWeave-Actor-ID"))
	if !domain.ValidCustomerID(subject) || strings.TrimSpace(headers.Get("X-StreamWeave-Actor-Type")) != "human" {
		return apporders.Actor{}, apporders.ErrUnauthorized
	}
	var scopes []string
	for _, scope := range strings.Fields(headers.Get("X-StreamWeave-Scopes")) {
		if !scopePattern.MatchString(scope) {
			return apporders.Actor{}, apporders.ErrUnauthorized
		}
		scopes = append(scopes, scope)
	}
	return apporders.Actor{Subject: subject, Type: "human", Scopes: scopes}, nil
}

type Server struct {
	httpServer       *http.Server
	health           *apphealth.Service
	orders           *apporders.Service
	authenticator    Authenticator
	logger           *slog.Logger
	openAPIPath      string
	readinessTimeout time.Duration
	requests         *prometheus.CounterVec
	durations        *prometheus.HistogramVec
	inFlight         prometheus.Gauge
	requestSequence  atomic.Uint64
}

// NewServer keeps the operations-only constructor used by foundation tests.
func NewServer(addr, metricsPath string, readinessTimeout time.Duration, health *apphealth.Service, logger *slog.Logger, openAPIPath string) (*Server, error) {
	return buildServer(addr, metricsPath, readinessTimeout, health, logger, openAPIPath, nil, nil)
}

func NewApplicationServer(addr, metricsPath string, readinessTimeout time.Duration, health *apphealth.Service, logger *slog.Logger, openAPIPath string, service *apporders.Service, authenticator Authenticator) (*Server, error) {
	if service == nil || authenticator == nil {
		return nil, errors.New("Orders application and authenticator are required")
	}
	return buildServer(addr, metricsPath, readinessTimeout, health, logger, openAPIPath, service, authenticator)
}

func buildServer(addr, metricsPath string, readinessTimeout time.Duration, health *apphealth.Service, logger *slog.Logger, openAPIPath string, service *apporders.Service, authenticator Authenticator) (*Server, error) {
	if health == nil {
		return nil, errors.New("Orders health service is required")
	}
	if metricsPath != "/metrics" {
		return nil, errors.New("Orders metrics path must be /metrics")
	}
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "orders", Name: "http_requests_total", Help: "Total Orders HTTP requests."}, []string{"route", "method", "status"})
	durations := prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "orders", Name: "http_request_duration_seconds", Help: "Orders HTTP request duration."}, []string{"route", "method"})
	inFlight := prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "orders", Name: "http_in_flight_requests", Help: "Orders HTTP requests currently in flight."})
	registry.MustRegister(requests, durations, inFlight)
	s := &Server{health: health, orders: service, authenticator: authenticator, logger: logger, openAPIPath: openAPIPath, readinessTimeout: readinessTimeout, requests: requests, durations: durations, inFlight: inFlight}
	mux := http.NewServeMux()
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	mux.HandleFunc(metricsPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			s.problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		metricsHandler.ServeHTTP(w, r)
	})
	mux.HandleFunc("/", s.route)
	s.httpServer = &http.Server{Addr: addr, Handler: s.observe(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	return s, nil
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
	switch {
	case r.URL.Path == "/health/live":
		s.method(w, r, http.MethodGet, func() { s.json(w, http.StatusOK, s.health.Live()) })
	case r.URL.Path == "/health/ready":
		s.method(w, r, http.MethodGet, func() {
			ctx, cancel := context.WithTimeout(r.Context(), s.readinessTimeout)
			defer cancel()
			status := s.health.Ready(ctx)
			code := http.StatusOK
			if status.Status != "ready" {
				code = http.StatusServiceUnavailable
			}
			s.json(w, code, status)
		})
	case r.URL.Path == "/v1/docs":
		s.method(w, r, http.MethodGet, func() { s.docs(w, r) })
	case r.URL.Path == "/v1/openapi.yaml":
		s.method(w, r, http.MethodGet, func() { s.openapi(w, r) })
	case r.URL.Path == "/v1/orders":
		if r.Method == http.MethodPost {
			s.createOrder(w, r)
			return
		}
		if r.Method == http.MethodGet {
			s.listOrders(w, r)
			return
		}
		s.problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	case strings.HasPrefix(r.URL.Path, "/v1/orders/") && strings.HasSuffix(r.URL.Path, "/cancel"):
		if r.Method == http.MethodPost {
			s.cancelOrder(w, r)
			return
		}
		s.problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	case strings.HasPrefix(r.URL.Path, "/v1/orders/"):
		if r.Method == http.MethodGet {
			s.getOrder(w, r)
			return
		}
		s.problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	default:
		s.problem(w, r, http.StatusNotFound, "not_found", "Route not found")
	}
}

func (s *Server) method(w http.ResponseWriter, r *http.Request, expected string, fn func()) {
	if r.Method != expected {
		s.problem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	fn()
}

type createRequest struct {
	CustomerID string `json:"customer_id"`
	Currency   string `json:"currency"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int32  `json:"quantity"`
	} `json:"items"`
}
type cancelRequest struct {
	ReasonCode string `json:"reason_code"`
}

func (s *Server) actor(r *http.Request) (apporders.Actor, error) {
	if s.authenticator == nil {
		return apporders.Actor{}, apporders.ErrUnauthorized
	}
	return s.authenticator.Authenticate(r.Context(), r.Header)
}
func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	actor, err := s.actor(r)
	if err != nil {
		s.problem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	key, ok := validIdempotencyKey(r.Header.Get("Idempotency-Key"))
	if !ok {
		s.problem(w, r, http.StatusBadRequest, "bad_request", "Idempotency-Key is invalid")
		return
	}
	var input createRequest
	if err := decodeJSON(r, &input); err != nil {
		s.problem(w, r, http.StatusBadRequest, "bad_request", "Request body is invalid")
		return
	}
	items := make([]domain.Item, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, domain.Item{ProductID: item.ProductID, Quantity: item.Quantity})
	}
	result, err := s.orders.Create(r.Context(), apporders.CreateInput{Actor: actor, CustomerID: input.CustomerID, Currency: input.Currency, Items: items, IdempotencyKey: key})
	if err != nil {
		s.writeApplicationError(w, r, err)
		return
	}
	status := result.ResponseStatus
	if status == "" {
		status = string(result.Order.Status)
	}
	createdAt := result.ResponseAt
	if createdAt.IsZero() {
		createdAt = result.Order.CreatedAt
	}
	s.json(w, http.StatusAccepted, acceptedResponse{OrderID: result.Order.ID, Status: status, CreatedAt: createdAt})
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	actor, err := s.actor(r)
	if err != nil {
		s.problem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	orderID := strings.TrimPrefix(r.URL.Path, "/v1/orders/")
	order, err := s.orders.Get(r.Context(), actor, orderID)
	if err != nil {
		s.writeApplicationError(w, r, err)
		return
	}
	s.json(w, http.StatusOK, toOrderResponse(order))
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	actor, err := s.actor(r)
	if err != nil {
		s.problem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	limit := 25
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			s.problem(w, r, http.StatusBadRequest, "bad_request", "limit is invalid")
			return
		}
	}
	orders, next, err := s.orders.List(r.Context(), apporders.ListInput{Actor: actor, CustomerID: r.URL.Query().Get("customer_id"), Status: r.URL.Query().Get("status"), Cursor: r.URL.Query().Get("cursor"), Limit: limit})
	if err != nil {
		s.writeApplicationError(w, r, err)
		return
	}
	items := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		items = append(items, toOrderResponse(order))
	}
	s.json(w, http.StatusOK, map[string]any{"items": items, "next_cursor": nullableString(next)})
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	actor, err := s.actor(r)
	if err != nil {
		s.problem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	key, ok := validIdempotencyKey(r.Header.Get("Idempotency-Key"))
	if !ok {
		s.problem(w, r, http.StatusBadRequest, "bad_request", "Idempotency-Key is invalid")
		return
	}
	path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/orders/"), "/cancel")
	reason := ""
	if r.ContentLength != 0 {
		var input cancelRequest
		if err := decodeJSON(r, &input); err != nil {
			s.problem(w, r, http.StatusBadRequest, "bad_request", "Request body is invalid")
			return
		}
		reason = input.ReasonCode
	}
	order, err := s.orders.Cancel(r.Context(), apporders.CancelInput{Actor: actor, OrderID: path, Reason: reason, IdempotencyKey: key})
	if err != nil {
		s.writeApplicationError(w, r, err)
		return
	}
	s.json(w, http.StatusAccepted, acceptedResponse{OrderID: order.ID, Status: string(order.Status), CreatedAt: order.UpdatedAt})
}

type acceptedResponse struct {
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
type orderResponse struct {
	OrderID     string         `json:"order_id"`
	CustomerID  string         `json:"customer_id"`
	Status      string         `json:"status"`
	Currency    string         `json:"currency"`
	TotalMinor  *int64         `json:"total_minor"`
	FailureCode *string        `json:"failure_code"`
	Items       []itemResponse `json:"items"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
type itemResponse struct {
	ProductID      string  `json:"product_id"`
	Quantity       int32   `json:"quantity"`
	UnitPriceMinor *int64  `json:"unit_price_minor"`
	LineTotalMinor *int64  `json:"line_total_minor"`
	QuoteID        *string `json:"quote_id"`
	QuoteVersion   *int64  `json:"quote_version"`
}

func toOrderResponse(order domain.Order) orderResponse {
	items := make([]itemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, itemResponse{ProductID: item.ProductID, Quantity: item.Quantity, UnitPriceMinor: item.UnitPriceMinor, LineTotalMinor: item.LineTotalMinor, QuoteID: item.QuoteID, QuoteVersion: item.QuoteVersion})
	}
	return orderResponse{OrderID: order.ID, CustomerID: order.CustomerID, Status: string(order.Status), Currency: order.Currency, TotalMinor: order.TotalMinor, FailureCode: order.FailureCode, Items: items, CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt}
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func validIdempotencyKey(value string) (string, bool) {
	value = strings.TrimSpace(value)
	return value, regexp.MustCompile(`^[A-Za-z0-9._:-]{16,256}$`).MatchString(value)
}

var scopePattern = regexp.MustCompile(`^[a-z0-9:_-]{2,96}$`)

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return errors.New("multiple JSON values")
	}
	return nil
}

func (s *Server) docs(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(s.openAPIPath); err != nil {
		s.problem(w, r, http.StatusNotFound, "not_found", "OpenAPI document unavailable")
		return
	}
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		s.problem(w, r, http.StatusInternalServerError, "internal_error", "Unable to initialize API documentation")
		return
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src https://unpkg.com 'unsafe-inline'; script-src https://unpkg.com 'nonce-"+nonce+"'; img-src https://unpkg.com data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>StreamWeave Orders API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script nonce="`+nonce+`">window.onload=()=>window.ui=SwaggerUIBundle({url:'/v1/openapi.yaml',dom_id:'#swagger-ui',deepLinking:true,displayRequestDuration:true,persistAuthorization:false,filter:true,tryItOutEnabled:true});</script></body></html>`)
}
func (s *Server) openapi(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(s.openAPIPath)
	if err != nil {
		s.problem(w, r, http.StatusNotFound, "not_found", "OpenAPI document unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
func (s *Server) json(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func (s *Server) problem(w http.ResponseWriter, r *http.Request, code int, problemType, detail string) {
	requestID := w.Header().Get("X-Request-ID")
	if r != nil && requestID == "" {
		requestID = r.Header.Get("X-Request-ID")
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://streamweave.dev/problems/" + problemType, "title": http.StatusText(code), "status": code, "detail": detail, "instance": r.URL.Path, "request_id": requestID})
}
func (s *Server) writeApplicationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, apporders.ErrUnauthorized):
		s.problem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication is required")
	case errors.Is(err, apporders.ErrForbidden):
		s.problem(w, r, http.StatusForbidden, "forbidden", "The requested operation is not permitted")
	case errors.Is(err, apporders.ErrNotFound):
		s.problem(w, r, http.StatusNotFound, "not_found", "Order was not found")
	case errors.Is(err, apporders.ErrIdempotencyConflict):
		s.problem(w, r, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was reused with different input")
	case errors.Is(err, apporders.ErrConflict):
		s.problem(w, r, http.StatusConflict, "state_conflict", "The order cannot be changed in its current state")
	case errors.Is(err, apporders.ErrInvalidCursor):
		s.problem(w, r, http.StatusBadRequest, "bad_request", "The cursor is invalid")
	case errors.Is(err, apporders.ErrDependency):
		s.problem(w, r, http.StatusServiceUnavailable, "dependency_unavailable", "A required dependency is unavailable")
	default:
		s.problem(w, r, http.StatusUnprocessableEntity, "validation_failed", "The request is invalid")
	}
}

func (s *Server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		route := routeTemplate(r.URL.Path)
		traceContext, span := otel.Tracer("streamweave/orders/http").Start(r.Context(), r.Method+" "+route)
		defer span.End()
		r = r.WithContext(traceContext)
		s.inFlight.Inc()
		defer s.inFlight.Dec()
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 {
			requestID = s.newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		s.requests.WithLabelValues(route, r.Method, strconv.Itoa(wrapped.status)).Inc()
		s.durations.WithLabelValues(route, r.Method).Observe(time.Since(started).Seconds())
		s.logger.Info("http request", "service", "orders", "method", r.Method, "route", route, "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds(), "request_id", requestID, "trace_id", span.SpanContext().TraceID().String())
	})
}
func routeTemplate(path string) string {
	switch {
	case path == "/health/live", path == "/health/ready", path == "/metrics", path == "/v1/docs", path == "/v1/openapi.yaml", path == "/v1/orders":
		return path
	case strings.HasSuffix(path, "/cancel") && strings.HasPrefix(path, "/v1/orders/"):
		return "/v1/orders/{order_id}/cancel"
	case strings.HasPrefix(path, "/v1/orders/"):
		return "/v1/orders/{order_id}"
	default:
		return "unmatched"
	}
}
func (s *Server) newRequestID() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err == nil {
		return "ord-" + hex.EncodeToString(bytes)
	}
	return "ord-" + time.Now().UTC().Format("20060102150405.000000000") + "-" + strconv.FormatUint(s.requestSequence.Add(1), 10)
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
