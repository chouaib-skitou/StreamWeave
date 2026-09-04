package upstreamhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domain "github.com/chouaib-skitou/streamweave/apps/gateway/internal/domain/gateway"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestClientForwardsSafeInternalContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer svc" || r.Header.Get("X-StreamWeave-Actor-ID") != "u1" || r.Header.Get("Idempotency-Key") != "key" || r.Header.Get("traceparent") == "" {
			t.Errorf("internal headers missing: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Internal-Topology", "secret")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: [16]byte{1}, SpanID: [8]byte{2}, TraceFlags: trace.FlagsSampled, Remote: true}))
	client, err := NewClient(map[string]string{"orders": server.URL}, server.Client(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	actor := &domain.Actor{Subject: "u1", Type: "human", Scopes: []string{"orders:read:self"}}
	response, err := client.Forward(ctx, "orders", "svc", actor, domain.ForwardRequest{Method: http.MethodGet, Path: "/v1/orders", RequestID: "r", CorrelationID: "c", IdempotencyKey: "key"})
	if err != nil || response.StatusCode != http.StatusAccepted || len(response.Header["Content-Type"]) != 1 {
		t.Fatalf("response=%+v err=%v", response, err)
	}
}

type retryTransport struct{ calls int }

func (t *retryTransport) RoundTrip(*http.Request) (*http.Response, error) {
	t.calls++
	if t.calls == 1 {
		return nil, errors.New("temporary transport failure")
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}, nil
}

func TestClientRetriesOnlySafeReads(t *testing.T) {
	transport := &retryTransport{}
	client, err := NewClient(map[string]string{"orders": "http://orders.internal"}, &http.Client{Transport: transport}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Forward(context.Background(), "orders", "svc", nil, domain.ForwardRequest{Method: http.MethodGet, Path: "/v1/orders"}); err != nil || transport.calls != 2 {
		t.Fatalf("safe read retry calls=%d err=%v", transport.calls, err)
	}
	transport.calls = 0
	if _, err := client.Forward(context.Background(), "orders", "svc", nil, domain.ForwardRequest{Method: http.MethodPost, Path: "/v1/orders"}); err == nil || transport.calls != 1 {
		t.Fatalf("mutation retry calls=%d err=%v", transport.calls, err)
	}
}

func TestClientMapsFailuresAndBoundsResponses(t *testing.T) {
	large := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("12345")) }))
	defer large.Close()
	client, _ := NewClient(map[string]string{"orders": large.URL}, large.Client(), 4)
	if _, err := client.Forward(context.Background(), "orders", "svc", nil, domain.ForwardRequest{Method: http.MethodGet, Path: "/x"}); err != domain.ErrUpstreamResponse {
		t.Fatalf("large response: %v", err)
	}
	if _, err := client.Forward(context.Background(), "missing", "svc", nil, domain.ForwardRequest{Method: http.MethodGet, Path: "/x"}); err != domain.ErrUpstreamUnavailable {
		t.Fatalf("missing owner: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	if _, err := client.Forward(ctx, "orders", "svc", nil, domain.ForwardRequest{Method: http.MethodGet, Path: "/x"}); err == nil {
		t.Fatal("cancelled request accepted")
	}
}
