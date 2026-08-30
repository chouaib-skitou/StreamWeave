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

type rateLimitRequest struct {
	key    string
	limit  int
	window time.Duration
}

func NewRateLimiter(client *Client, login, refresh, reset, service int) *RateLimiter {
	return NewRateLimiterWithSources(client, login, refresh, reset, service, login*6, refresh*4, reset*4, service*4)
}

func NewRateLimiterWithSources(client *Client, login, refresh, reset, service, loginSource, refreshSource, resetSource, serviceSource int) *RateLimiter {
	return &RateLimiter{client: client, limits: map[string]Limit{
		"login-account":         {Max: login, Window: 15 * time.Minute},
		"login-source":          {Max: loginSource, Window: 15 * time.Minute},
		"refresh-account":       {Max: refresh, Window: time.Minute},
		"refresh-source":        {Max: refreshSource, Window: time.Minute},
		"reset-account":         {Max: reset, Window: time.Hour},
		"reset-source":          {Max: resetSource, Window: time.Hour},
		"service-token-account": {Max: service, Window: time.Minute},
		"service-token-source":  {Max: serviceSource, Window: time.Minute},
		"register-account":      {Max: login, Window: 15 * time.Minute},
		"register-source":       {Max: loginSource, Window: 15 * time.Minute},
		"login-global":          {Max: loginSource * 10, Window: 15 * time.Minute},
		"register-global":       {Max: loginSource * 10, Window: 15 * time.Minute},
		"refresh-global":        {Max: refreshSource * 10, Window: time.Minute},
		"reset-global":          {Max: resetSource * 10, Window: time.Hour},
		"service-token-global":  {Max: serviceSource * 10, Window: time.Minute},
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

func (r *RateLimiter) AllowMany(ctx context.Context, keys []string) (bool, error) {
	requests := make([]rateLimitRequest, 0, len(keys))
	for _, key := range keys {
		kind := key
		for i, char := range key {
			if char == ':' {
				kind = key[:i]
				break
			}
		}
		limit, ok := r.limits[kind]
		if !ok {
			limit = Limit{Max: 30, Window: time.Minute}
		}
		requests = append(requests, rateLimitRequest{key: key, limit: limit.Max, window: limit.Window})
	}
	return r.client.AllowMany(ctx, requests)
}
