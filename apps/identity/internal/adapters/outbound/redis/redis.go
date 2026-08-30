package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct{ client *redis.Client }

var allowManyScript = redis.NewScript(`
local allowed = 1
for index, key in ipairs(KEYS) do
  local count = redis.call('INCR', key)
  if count == 1 then redis.call('EXPIRE', key, ARGV[index + #KEYS]) end
  if count > tonumber(ARGV[index]) then allowed = 0 end
end
return allowed
`)

func (c *Client) Name() string { return "redis" }

func Open(rawURL string) (*Client, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}
	return &Client{client: redis.NewClient(options)}, nil
}

func (c *Client) Check(ctx context.Context) error { return c.client.Ping(ctx).Err() }
func (c *Client) Close() error                    { return c.client.Close() }

func (c *Client) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	key = "identity:rate:" + digest(key)
	pipe := c.client.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	return count.Val() <= int64(limit), nil
}

func (c *Client) AllowMany(ctx context.Context, requests []rateLimitRequest) (bool, error) {
	if len(requests) == 0 {
		return true, nil
	}
	keys := make([]string, len(requests))
	args := make([]any, 0, len(requests)*2)
	for index, request := range requests {
		keys[index] = "identity:rate:" + digest(request.key)
		args = append(args, request.limit)
	}
	for _, request := range requests {
		args = append(args, int(request.window.Seconds()))
	}
	result, err := allowManyScript.Run(ctx, c.client, keys, args...).Int()
	return result == 1, err
}

func (c *Client) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return c.client.Set(ctx, "identity:revoked:"+digest(jti), "1", ttl).Err()
}

func (c *Client) IsRevoked(ctx context.Context, jti string) (bool, error) {
	value, err := c.client.Exists(ctx, "identity:revoked:"+digest(jti)).Result()
	return value == 1, err
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
