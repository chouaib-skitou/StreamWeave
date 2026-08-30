package httpadapter

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPageQueryDefaultsAndCursorRoundTrip(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/v1/users?email_prefix=Alice", nil)
	if err != nil {
		t.Fatal(err)
	}
	query, err := parsePageQuery(request)
	if err != nil {
		t.Fatal(err)
	}
	if query.limit != defaultPageSize || query.emailPrefix != "alice" {
		t.Fatalf("unexpected query: %+v", query)
	}
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	id := uuid.New()
	encoded := encodeCursor(createdAt, id)
	decodedTime, decodedID, err := decodeCursor(encoded)
	if err != nil || !decodedTime.Equal(createdAt) || decodedID != id {
		t.Fatalf("cursor mismatch: %v %v %v", decodedTime, decodedID, err)
	}
}

func TestPageQueryRejectsUnsafeBoundsAndCursors(t *testing.T) {
	for _, target := range []string{"/v1/users?limit=0", "/v1/users?limit=101", "/v1/users?limit=nope", "/v1/users?cursor=bad", "/v1/users?email_prefix=%0A"} {
		request, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parsePageQuery(request); err == nil {
			t.Fatalf("accepted %s", target)
		}
	}
}

func TestPageQueryEscapesEmailFilterWildcards(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/v1/users?email_prefix=a%25_b", nil)
	if err != nil {
		t.Fatal(err)
	}
	query, err := parsePageQuery(request)
	if err != nil {
		t.Fatal(err)
	}
	if query.emailPrefix != `a\%\_b` {
		t.Fatalf("email prefix was not escaped: %q", query.emailPrefix)
	}
}

func TestRequestSourceUsesCoarsePrefixesAndTrustedForwarding(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.RemoteAddr = "192.0.2.44:1234"
	if got := requestSource(request, false); got != "192.0.2.0/24" {
		t.Fatalf("source: %s", got)
	}
	request.Header.Set("X-Forwarded-For", "198.51.100.77, 192.0.2.44")
	if got := requestSource(request, false); got != "192.0.2.0/24" {
		t.Fatalf("untrusted source: %s", got)
	}
	if got := requestSource(request, true); got != "198.51.100.0/24" {
		t.Fatalf("trusted source: %s", got)
	}
}
