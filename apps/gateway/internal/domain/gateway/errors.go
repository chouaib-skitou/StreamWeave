package gateway

import "errors"

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrRateLimited         = errors.New("rate limited")
	ErrDependency          = errors.New("dependency unavailable")
	ErrBadRequest          = errors.New("bad request")
	ErrPayloadTooLarge     = errors.New("payload too large")
	ErrUnsupportedMedia    = errors.New("unsupported media type")
	ErrUpstreamTimeout     = errors.New("upstream timeout")
	ErrUpstreamUnavailable = errors.New("upstream unavailable")
	ErrUpstreamResponse    = errors.New("invalid upstream response")
)

type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	RequestID string `json:"request_id"`
}
