package upstreamhttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	domain "github.com/chouaib-skitou/streamweave/apps/gateway/internal/domain/gateway"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type Client struct {
	URLs            map[string]string
	HTTPClient      *http.Client
	MaxResponseSize int64
	MaxInFlight     int
	inFlight        chan struct{}
}

func NewClient(urls map[string]string, httpClient *http.Client, maxResponseSize int64) (*Client, error) {
	if len(urls) == 0 {
		return nil, errors.New("upstream URLs are required")
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	if httpClient.CheckRedirect == nil {
		httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	}
	if maxResponseSize <= 0 {
		maxResponseSize = 4 << 20
	}
	const defaultMaxInFlight = 128
	return &Client{URLs: cloneURLs(urls), HTTPClient: httpClient, MaxResponseSize: maxResponseSize, MaxInFlight: defaultMaxInFlight, inFlight: make(chan struct{}, defaultMaxInFlight)}, nil
}

func (c *Client) Forward(ctx context.Context, owner, serviceToken string, actor *domain.Actor, input domain.ForwardRequest) (*domain.Response, error) {
	if strings.TrimSpace(serviceToken) == "" {
		return nil, domain.ErrUpstreamUnavailable
	}
	base, ok := c.URLs[owner]
	if !ok {
		return nil, domain.ErrUpstreamUnavailable
	}
	target, err := url.JoinPath(strings.TrimRight(base, "/"), input.Path)
	if err != nil {
		return nil, domain.ErrUpstreamUnavailable
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, domain.ErrUpstreamUnavailable
	}
	parsed.RawQuery = input.RawQuery
	select {
	case c.inFlight <- struct{}{}:
		defer func() { <-c.inFlight }()
	case <-ctx.Done():
		return nil, classifyContextError(ctx)
	}
	for attempt := 0; attempt < 2; attempt++ {
		request, err := http.NewRequestWithContext(ctx, input.Method, parsed.String(), bytes.NewReader(input.Body))
		if err != nil {
			return nil, domain.ErrBadRequest
		}
		request.Header.Set("Authorization", "Bearer "+serviceToken)
		request.Header.Set("X-Request-ID", input.RequestID)
		request.Header.Set("X-Correlation-ID", input.CorrelationID)
		if input.ContentType != "" {
			request.Header.Set("Content-Type", input.ContentType)
		}
		if input.IdempotencyKey != "" {
			request.Header.Set("Idempotency-Key", input.IdempotencyKey)
		}
		if actor != nil {
			normalized := actor.Normalized()
			request.Header.Set("X-StreamWeave-Actor-ID", normalized.Subject)
			request.Header.Set("X-StreamWeave-Actor-Type", normalized.Type)
			request.Header.Set("X-StreamWeave-Scopes", strings.Join(normalized.Scopes, " "))
			if normalized.SessionID != "" {
				request.Header.Set("X-StreamWeave-Actor-Session-ID", normalized.SessionID)
			}
		}
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(request.Header))
		response, err := c.HTTPClient.Do(request)
		if err == nil {
			defer response.Body.Close()
			body, readErr := io.ReadAll(io.LimitReader(response.Body, c.MaxResponseSize+1))
			if readErr != nil || int64(len(body)) > c.MaxResponseSize {
				return nil, domain.ErrUpstreamResponse
			}
			return &domain.Response{StatusCode: response.StatusCode, Header: safeHeaders(response.Header), Body: body}, nil
		}
		if ctx.Err() != nil {
			return nil, classifyContextError(ctx)
		}
		if attempt == 0 && (input.Method == http.MethodGet || input.Method == http.MethodHead) {
			continue
		}
		return nil, domain.ErrUpstreamUnavailable
	}
	return nil, domain.ErrUpstreamUnavailable
}

func classifyContextError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return domain.ErrUpstreamTimeout
	}
	return domain.ErrUpstreamUnavailable
}

func cloneURLs(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func safeHeaders(input http.Header) map[string][]string {
	output := make(map[string][]string)
	for _, name := range []string{"Content-Type", "Cache-Control", "ETag", "Last-Modified", "Location", "Retry-After"} {
		if values, ok := input[http.CanonicalHeaderKey(name)]; ok {
			output[name] = append([]string(nil), values...)
		}
	}
	return output
}

func (c *Client) String() string { return fmt.Sprintf("upstreams=%d", len(c.URLs)) }
