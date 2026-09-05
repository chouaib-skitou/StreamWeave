package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Source interface{ Lookup(string) (string, bool) }
type OSEnv struct{}

func (OSEnv) Lookup(key string) (string, bool) { return os.LookupEnv(key) }

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	KafkaBrokers     string
	RedisURL         string
	Environment      string
	LogLevel         string
	MetricsPath      string
	OpenAPIPath      string
	ShutdownTimeout  time.Duration
	ReadinessTimeout time.Duration
	OTelEndpoint     string
	OTelInsecure     bool
}

func Load(source Source) (Config, error) {
	c := Config{
		HTTPAddr:         value(source, "ORDERS_HTTP_ADDR", ":8080"),
		DatabaseURL:      value(source, "ORDERS_DATABASE_URL", ""),
		KafkaBrokers:     value(source, "ORDERS_KAFKA_BROKERS", ""),
		RedisURL:         value(source, "ORDERS_REDIS_URL", ""),
		Environment:      value(source, "ORDERS_ENVIRONMENT", "development"),
		LogLevel:         value(source, "ORDERS_LOG_LEVEL", "info"),
		MetricsPath:      value(source, "ORDERS_METRICS_PATH", "/metrics"),
		OpenAPIPath:      value(source, "ORDERS_OPENAPI_PATH", "contracts/openapi/orders.openapi.yaml"),
		ShutdownTimeout:  duration(source, "ORDERS_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadinessTimeout: duration(source, "ORDERS_READINESS_TIMEOUT", 2*time.Second),
		OTelEndpoint:     value(source, "ORDERS_OTEL_ENDPOINT", ""),
		OTelInsecure:     boolean(source, "ORDERS_OTEL_INSECURE", false),
	}
	if raw, ok := source.Lookup("ORDERS_OTEL_INSECURE"); ok && strings.TrimSpace(raw) != "" {
		if _, err := strconv.ParseBool(raw); err != nil {
			return Config{}, errors.New("ORDERS_OTEL_INSECURE must be a boolean")
		}
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return errors.New("ORDERS_HTTP_ADDR must not be empty")
	}
	if c.Environment != "development" && c.Environment != "staging" && c.Environment != "production" {
		return errors.New("ORDERS_ENVIRONMENT must be development, staging, or production")
	}
	if c.MetricsPath != "/metrics" {
		return errors.New("ORDERS_METRICS_PATH must be /metrics")
	}
	if c.ShutdownTimeout <= 0 || c.ReadinessTimeout <= 0 {
		return errors.New("Orders timeouts must be positive")
	}
	if c.Environment != "development" {
		if strings.TrimSpace(c.DatabaseURL) == "" || strings.TrimSpace(c.KafkaBrokers) == "" {
			return errors.New("Orders database and Kafka configuration are required outside development")
		}
		if strings.TrimSpace(c.OTelEndpoint) == "" || c.OTelInsecure {
			return errors.New("secure Orders OpenTelemetry configuration is required outside development")
		}
		if !tlsPostgres(c.DatabaseURL) {
			return errors.New("ORDERS_DATABASE_URL must require PostgreSQL TLS outside development")
		}
	}
	return nil
}

func (c Config) SafeSummary() string {
	return fmt.Sprintf("environment=%s http_addr=%s", c.Environment, c.HTTPAddr)
}

func tlsPostgres(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	mode := strings.ToLower(u.Query().Get("sslmode"))
	return mode == "require" || mode == "verify-ca" || mode == "verify-full"
}
func value(source Source, key, fallback string) string {
	if v, ok := source.Lookup(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
func duration(source Source, key string, fallback time.Duration) time.Duration {
	v, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return -1
	}
	return d
}
func boolean(source Source, key string, fallback bool) bool {
	v, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}
