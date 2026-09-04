package redisadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Class struct {
	Limit       int
	Window      time.Duration
	Burst       int
	BurstWindow time.Duration
}

var classes = map[string]Class{
	"auth-write":         {Limit: 10, Window: time.Minute, Burst: 5, BurstWindow: time.Second},
	"demo-registration":  {Limit: 5, Window: time.Minute, Burst: 2, BurstWindow: time.Second},
	"authenticated-read": {Limit: 120, Window: time.Minute, Burst: 30, BurstWindow: time.Second},
	"order-read":         {Limit: 120, Window: time.Minute, Burst: 30, BurstWindow: time.Second},
	"order-write":        {Limit: 30, Window: time.Minute, Burst: 10, BurstWindow: time.Second},
}

var allowScript = redis.NewScript(`
local first = redis.call('INCR', KEYS[1])
if first == 1 then redis.call('EXPIRE', KEYS[1], ARGV[3]) end
local second = redis.call('INCR', KEYS[2])
if second == 1 then redis.call('EXPIRE', KEYS[2], ARGV[4]) end
if first > tonumber(ARGV[1]) or second > tonumber(ARGV[2]) then return 0 end
return 1
`)

type Client struct {
	client *redis.Client
	prefix string
}

func Open(rawURL string) (*Client, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse gateway redis URL: %w", err)
	}
	return &Client{client: redis.NewClient(options), prefix: "streamweave:gateway:ratelimit:v1"}, nil
}
func New(client *redis.Client) *Client {
	return &Client{client: client, prefix: "streamweave:gateway:ratelimit:v1"}
}
func (c *Client) Check(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("redis client is unavailable")
	}
	return c.client.Ping(ctx).Err()
}
func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Client) Allow(ctx context.Context, key, className string) (bool, error) {
	if className == "health" || className == "metrics" {
		return true, nil
	}
	class, ok := classes[className]
	if !ok {
		return false, fmt.Errorf("unknown gateway rate-limit class %q", className)
	}
	if err := c.Check(ctx); err != nil {
		return false, err
	}
	digest := digest(key)
	now := time.Now().UTC()
	windowKey := fmt.Sprintf("%s:%s:%d", c.prefix, digest, now.Unix()/int64(class.Window.Seconds()))
	burstKey := fmt.Sprintf("%s:burst:%s:%d", c.prefix, digest, now.Unix()/int64(class.BurstWindow.Seconds()))
	result, err := allowScript.Run(ctx, c.client, []string{windowKey, burstKey}, class.Limit, class.Burst, int(class.Window.Seconds()), int(class.BurstWindow.Seconds())).Int()
	return result == 1, err
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
