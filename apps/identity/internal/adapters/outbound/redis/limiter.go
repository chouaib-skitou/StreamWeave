package redis

import (
	"context"
	"time"
)

type RateLimiter struct {
	client *Client
	limits map[string]Limit
}

type Limit struct {
	Max    int
	Window time.Duration
}

func NewRateLimiter(client *Client, login, refresh, reset, service int) *RateLimiter {
	return &RateLimiter{client: client, limits: map[string]Limit{
		"login":         {Max: login, Window: 15 * time.Minute},
		"refresh":       {Max: refresh, Window: time.Minute},
		"reset":         {Max: reset, Window: time.Hour},
		"service-token": {Max: service, Window: time.Minute},
		"register":      {Max: login, Window: 15 * time.Minute},
	}}
}

func (r *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	kind := key
	if index := len(key); index > 0 {
		for i, char := range key {
			if char == ':' {
				kind = key[:i]
				break
			}
		}
	}
	limit, ok := r.limits[kind]
	if !ok {
		limit = Limit{Max: 30, Window: time.Minute}
	}
	return r.client.Allow(ctx, key, limit.Max, limit.Window)
}
