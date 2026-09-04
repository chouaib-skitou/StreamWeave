package identityclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientCachesServiceTokenAndRejectsBadResponses(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/auth/service-token" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"svc-token","expires_in":300}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "gateway", "secret", []string{"orders:read"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.Token(context.Background())
	if err != nil || first != "svc-token" {
		t.Fatalf("first=%q err=%v", first, err)
	}
	second, err := client.Token(context.Background())
	if err != nil || second != first || calls != 1 {
		t.Fatalf("cached=%q err=%v calls=%d", second, err, calls)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer bad.Close()
	invalid, _ := NewClient(bad.URL, "gateway", "secret", []string{"orders:read"}, bad.Client())
	if _, err := invalid.Token(context.Background()); err == nil {
		t.Fatal("bad status accepted")
	}
	if _, err := NewClient(server.URL, "", "secret", []string{"x"}, nil); err == nil {
		t.Fatal("missing id accepted")
	}
	_ = strings.TrimSpace
	_ = time.Second
}
