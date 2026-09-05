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
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	apphealth "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/health"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	httpServer       *http.Server
	health           *apphealth.Service
	logger           *slog.Logger
	openAPIPath      string
	readinessTimeout time.Duration
	requests         *prometheus.CounterVec
	durations        *prometheus.HistogramVec
	inFlight         prometheus.Gauge
	requestSequence  atomic.Uint64
}

func NewServer(addr, metricsPath string, readinessTimeout time.Duration, health *apphealth.Service, logger *slog.Logger, openAPIPath string) (*Server, error) {
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
	s := &Server{health: health, logger: logger, openAPIPath: openAPIPath, readinessTimeout: readinessTimeout, requests: requests, durations: durations, inFlight: inFlight}
	mux := http.NewServeMux()
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	mux.HandleFunc(metricsPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			s.problem(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
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
	switch r.URL.Path {
	case "/health/live":
		if r.Method != http.MethodGet {
			s.problem(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		s.json(w, http.StatusOK, s.health.Live())
	case "/health/ready":
		if r.Method != http.MethodGet {
			s.problem(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), s.readinessTimeout)
		defer cancel()
		status := s.health.Ready(ctx)
		code := http.StatusOK
		if status.Status != "ready" {
			code = http.StatusServiceUnavailable
		}
		s.json(w, code, status)
	case "/v1/docs":
		if r.Method != http.MethodGet {
			s.problem(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		s.docs(w)
	case "/v1/openapi.yaml":
		if r.Method != http.MethodGet {
			s.problem(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}
		s.openapi(w)
	default:
		s.problem(w, http.StatusNotFound, "not_found", "Route not found")
	}
}

func (s *Server) docs(w http.ResponseWriter) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		s.problem(w, http.StatusInternalServerError, "internal_error", "Unable to initialize API documentation")
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

func (s *Server) openapi(w http.ResponseWriter) {
	data, err := os.ReadFile(s.openAPIPath)
	if err != nil {
		s.problem(w, http.StatusServiceUnavailable, "dependency_unavailable", "OpenAPI document unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(data)
}

func (s *Server) json(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func (s *Server) problem(w http.ResponseWriter, code int, problemType, detail string) {
	requestID := w.Header().Get("X-Request-ID")
	s.json(w, code, map[string]any{"type": "https://streamweave.dev/problems/" + problemType, "title": http.StatusText(code), "status": code, "detail": detail, "instance": requestID, "request_id": requestID})
}

func (s *Server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		s.inFlight.Inc()
		defer s.inFlight.Dec()
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 {
			requestID = s.newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		route := r.URL.Path
		s.requests.WithLabelValues(route, r.Method, strconv.Itoa(wrapped.status)).Inc()
		s.durations.WithLabelValues(route, r.Method).Observe(time.Since(started).Seconds())
		s.logger.Info("http request", "service", "orders", "method", r.Method, "route", route, "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds(), "request_id", requestID)
	})
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
