package gateway

import (
	"context"
	"errors"
	"net/http"
	"testing"

	domain "github.com/chouaib-skitou/streamweave/apps/gateway/internal/domain/gateway"
)

type verifierStub struct {
	actor domain.Actor
	err   error
}

func (v verifierStub) Verify(context.Context, string) (domain.Actor, error) { return v.actor, v.err }

type limiterStub struct {
	allowed bool
	err     error
	key     string
	class   string
}

func (l *limiterStub) Allow(_ context.Context, key, class string) (bool, error) {
	l.key, l.class = key, class
	return l.allowed, l.err
}
func (l *limiterStub) Check(context.Context) error { return nil }

type tokenStub struct {
	token string
	err   error
}

func (t tokenStub) Token(context.Context) (string, error)   { return t.token, t.err }
func (t tokenStub) Refresh(context.Context) (string, error) { return t.token, t.err }

type refreshErrorTokenStub struct{}

func (refreshErrorTokenStub) Token(context.Context) (string, error) { return "expired", nil }
func (refreshErrorTokenStub) Refresh(context.Context) (string, error) {
	return "", errors.New("refresh failed")
}

type nonRefreshableTokenStub struct{}

func (nonRefreshableTokenStub) Token(context.Context) (string, error) { return "svc", nil }

type sequenceTokenStub struct {
	first, second string
	refreshes     int
}

func (t *sequenceTokenStub) Token(context.Context) (string, error) { return t.first, nil }
func (t *sequenceTokenStub) Refresh(context.Context) (string, error) {
	t.refreshes++
	return t.second, nil
}

type sequenceUpstreamStub struct {
	responses []*domain.Response
	calls     int
	tokens    []string
}

func (u *sequenceUpstreamStub) Forward(_ context.Context, _ string, token string, _ *domain.Actor, _ domain.ForwardRequest) (*domain.Response, error) {
	u.calls++
	u.tokens = append(u.tokens, token)
	return u.responses[u.calls-1], nil
}

type upstreamStub struct {
	response *domain.Response
	err      error
	owner    string
	token    string
}

func (u *upstreamStub) Forward(_ context.Context, owner, token string, _ *domain.Actor, _ domain.ForwardRequest) (*domain.Response, error) {
	u.owner, u.token = owner, token
	return u.response, u.err
}

func TestNewServiceRequiresDependencies(t *testing.T) {
	if _, err := NewService(nil, nil, nil, nil); err == nil {
		t.Fatal("expected dependency error")
	}
}

func TestAuthorizePublicAndProtected(t *testing.T) {
	limiter := &limiterStub{allowed: true}
	service, err := NewService(verifierStub{actor: domain.Actor{Subject: "u1", Type: "human", Scopes: []string{"orders:read:self"}}}, limiter, tokenStub{token: "svc"}, &upstreamStub{})
	if err != nil {
		t.Fatal(err)
	}
	request := httptestRequest("GET", "/v1/orders", "")
	public := domain.RoutePolicy{Auth: domain.AuthNone, LimitClass: "health"}
	if actor, raw, err := service.Authorize(context.Background(), request, public, "source"); err != nil || raw != "" || actor.Subject != "" {
		t.Fatalf("public authorization: actor=%+v raw=%q err=%v", actor, raw, err)
	}
	protected := domain.RoutePolicy{Auth: domain.AuthBearer, Scopes: []string{"orders:read:self"}, LimitClass: "order-read"}
	request.Header.Set("Authorization", "Bearer human-token")
	actor, raw, err := service.Authorize(context.Background(), request, protected, "source")
	if err != nil || raw != "human-token" || actor.Subject != "u1" {
		t.Fatalf("protected authorization: actor=%+v raw=%q err=%v", actor, raw, err)
	}
}

func TestAuthorizeFailures(t *testing.T) {
	request := httptestRequest("GET", "/v1/orders", "")
	protected := domain.RoutePolicy{Auth: domain.AuthBearer, Scopes: []string{"orders:read:self"}, LimitClass: "order-read"}
	for name, limiter := range map[string]*limiterStub{
		"rate": {allowed: false}, "dependency": {err: errors.New("redis down")},
	} {
		service, _ := NewService(verifierStub{}, limiter, tokenStub{}, &upstreamStub{})
		if _, _, err := service.Authorize(context.Background(), request, protected, "source"); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
	service, _ := NewService(verifierStub{}, &limiterStub{allowed: true}, tokenStub{}, &upstreamStub{})
	if _, _, err := service.Authorize(context.Background(), request, protected, "source"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("missing bearer: %v", err)
	}
	request.Header.Set("Authorization", "Bearer token")
	service, _ = NewService(verifierStub{err: errors.New("bad")}, &limiterStub{allowed: true}, tokenStub{}, &upstreamStub{})
	if _, _, err := service.Authorize(context.Background(), request, protected, "source"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("bad token: %v", err)
	}
	service, _ = NewService(verifierStub{actor: domain.Actor{Subject: "u", Type: "human", Scopes: []string{"other"}}}, &limiterStub{allowed: true}, tokenStub{}, &upstreamStub{})
	if _, _, err := service.Authorize(context.Background(), request, protected, "source"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("scope: %v", err)
	}
}

func TestAuthorizeRejectsInvalidInputAndLimiterFailures(t *testing.T) {
	service, _ := NewService(verifierStub{}, &limiterStub{allowed: true}, tokenStub{}, &upstreamStub{})
	if _, _, err := service.Authorize(context.Background(), nil, domain.RoutePolicy{}, "source"); !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("nil request: %v", err)
	}

	request := httptestRequest("GET", "/v1/orders", "Bearer token")
	protected := domain.RoutePolicy{Auth: domain.AuthBearer, Scopes: []string{"orders:read:self"}, LimitClass: "order-read"}
	invalidActor := verifierStub{actor: domain.Actor{Subject: "u1", Type: "service", Scopes: []string{"orders:read:self"}}}
	service, _ = NewService(invalidActor, &limiterStub{allowed: true}, tokenStub{}, &upstreamStub{})
	if _, _, err := service.Authorize(context.Background(), request, protected, "source"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("invalid actor: %v", err)
	}
	validActor := verifierStub{actor: domain.Actor{Subject: "u1", Type: "human", Scopes: []string{"orders:read:self"}}}
	service, _ = NewService(validActor, &limiterStub{allowed: false}, tokenStub{}, &upstreamStub{})
	if _, _, err := service.Authorize(context.Background(), request, protected, "source"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("rate limit: %v", err)
	}
	service, _ = NewService(validActor, &limiterStub{err: errors.New("redis down")}, tokenStub{}, &upstreamStub{})
	if _, _, err := service.Authorize(context.Background(), request, protected, "source"); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("limiter dependency: %v", err)
	}
}

func TestForwardMapsDependenciesAndResponses(t *testing.T) {
	upstream := &upstreamStub{response: &domain.Response{StatusCode: 202}}
	service, _ := NewService(verifierStub{}, &limiterStub{allowed: true}, tokenStub{token: "svc"}, upstream)
	response, err := service.Forward(context.Background(), "orders", nil, domain.ForwardRequest{Path: "/v1/orders", RequestID: "r", CorrelationID: "c", IdempotencyKey: "i"}, domain.RoutePolicy{})
	if err != nil || response.StatusCode != 202 || upstream.owner != "orders" || upstream.token != "svc" {
		t.Fatalf("forward: response=%+v err=%v", response, err)
	}
	service, _ = NewService(verifierStub{}, &limiterStub{allowed: true}, tokenStub{err: errors.New("token")}, upstream)
	if _, err := service.Forward(context.Background(), "orders", nil, domain.ForwardRequest{Path: "/v1/orders"}, domain.RoutePolicy{}); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("token dependency: %v", err)
	}
	service, _ = NewService(verifierStub{}, &limiterStub{allowed: true}, tokenStub{token: "svc"}, &upstreamStub{err: domain.ErrUpstreamTimeout})
	if _, err := service.Forward(context.Background(), "orders", nil, domain.ForwardRequest{Path: "/v1/orders"}, domain.RoutePolicy{}); !errors.Is(err, domain.ErrUpstreamTimeout) {
		t.Fatalf("upstream: %v", err)
	}
	service, _ = NewService(verifierStub{}, &limiterStub{allowed: true}, tokenStub{token: "svc"}, &upstreamStub{response: &domain.Response{StatusCode: 0}})
	if _, err := service.Forward(context.Background(), "orders", nil, domain.ForwardRequest{Path: "/v1/orders"}, domain.RoutePolicy{}); !errors.Is(err, domain.ErrUpstreamResponse) {
		t.Fatalf("invalid response: %v", err)
	}
}

func TestForwardRefreshesOnlySafeReadAfterServiceUnauthorized(t *testing.T) {
	tokens := &sequenceTokenStub{first: "expired", second: "fresh"}
	upstream := &sequenceUpstreamStub{responses: []*domain.Response{{StatusCode: 401}, {StatusCode: 200, Body: []byte("ok")}}}
	service, _ := NewService(verifierStub{}, &limiterStub{allowed: true}, tokens, upstream)
	response, err := service.Forward(context.Background(), "orders", nil, domain.ForwardRequest{Method: "GET", Path: "/v1/orders"}, domain.RoutePolicy{RetrySafeRead: true})
	if err != nil || response.StatusCode != 200 || upstream.calls != 2 || tokens.refreshes != 1 || upstream.tokens[1] != "fresh" {
		t.Fatalf("response=%+v calls=%d refreshes=%d tokens=%v err=%v", response, upstream.calls, tokens.refreshes, upstream.tokens, err)
	}
	mutationUpstream := &sequenceUpstreamStub{responses: []*domain.Response{{StatusCode: 401}}}
	service, _ = NewService(verifierStub{}, &limiterStub{allowed: true}, tokens, mutationUpstream)
	response, err = service.Forward(context.Background(), "orders", nil, domain.ForwardRequest{Method: "POST", Path: "/v1/orders"}, domain.RoutePolicy{Mutation: true})
	if err != nil || response.StatusCode != 401 || mutationUpstream.calls != 1 {
		t.Fatalf("mutation retry response=%+v calls=%d err=%v", response, mutationUpstream.calls, err)
	}
	service, _ = NewService(verifierStub{}, &limiterStub{allowed: true}, refreshErrorTokenStub{}, &sequenceUpstreamStub{responses: []*domain.Response{{StatusCode: 401}}})
	if _, err := service.Forward(context.Background(), "orders", nil, domain.ForwardRequest{Method: "GET", Path: "/v1/orders"}, domain.RoutePolicy{RetrySafeRead: true}); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("refresh dependency: %v", err)
	}
	service, _ = NewService(verifierStub{}, &limiterStub{allowed: true}, nonRefreshableTokenStub{}, &sequenceUpstreamStub{responses: []*domain.Response{{StatusCode: 401}}})
	if response, err := service.Forward(context.Background(), "orders", nil, domain.ForwardRequest{Method: "GET", Path: "/v1/orders"}, domain.RoutePolicy{RetrySafeRead: true}); err != nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("non-refreshable provider: response=%+v err=%v", response, err)
	}
}

func httptestRequest(method, path, auth string) *http.Request {
	req, _ := http.NewRequest(method, path, nil)
	req.Header.Set("Authorization", auth)
	return req
}
