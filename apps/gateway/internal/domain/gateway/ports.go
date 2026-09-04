package gateway

import (
	"context"
	"time"
)

type ClaimsVerifier interface {
	Verify(context.Context, string) (Actor, error)
}

type RateLimiter interface {
	Allow(context.Context, string, string) (bool, error)
	Check(context.Context) error
}

type ServiceTokenProvider interface {
	Token(context.Context) (string, error)
}

type RefreshableServiceTokenProvider interface {
	ServiceTokenProvider
	Refresh(context.Context) (string, error)
}

type UpstreamClient interface {
	Forward(context.Context, string, string, *Actor, ForwardRequest) (*Response, error)
}

type Metrics interface {
	ObserveAuthFailure(string, string)
	ObserveRateLimitRejection(string, string)
	ObserveRateLimiterError(string)
	ObserveJWKSRefresh(string)
	ObserveUpstreamRequest(string, string, string)
	ObserveUpstreamDuration(string, string, time.Duration)
}

type ForwardRequest struct {
	Method         string
	Path           string
	Route          string
	RawQuery       string
	Body           []byte
	ContentType    string
	RequestID      string
	CorrelationID  string
	IdempotencyKey string
}

type Response struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
}
