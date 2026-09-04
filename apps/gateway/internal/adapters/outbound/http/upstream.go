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
)

type Client struct {
	URLs            map[string]string
	HTTPClient      *http.Client
	MaxResponseSize int64
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
	return &Client{URLs: cloneURLs(urls), HTTPClient: httpClient, MaxResponseSize: maxResponseSize}, nil
}

func (c *Client) Forward(ctx context.Context, owner, serviceToken string, actor *domain.Actor, input domain.ForwardRequest) (*domain.Response, error) {
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
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, domain.ErrUpstreamTimeout
		}
		return nil, domain.ErrUpstreamUnavailable
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, c.MaxResponseSize+1))
	if err != nil {
		return nil, domain.ErrUpstreamResponse
	}
	if int64(len(body)) > c.MaxResponseSize {
		return nil, domain.ErrUpstreamResponse
	}
	return &domain.Response{StatusCode: response.StatusCode, Header: safeHeaders(response.Header), Body: body}, nil
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
