package gateway

import "context"

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

type ForwardRequest struct {
	Method         string
	Path           string
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
