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
	Role                string
	HTTPAddr            string
	DatabaseURL         string
	KafkaBrokers        string
	KafkaTLS            bool
	KafkaSASLUsername   string
	KafkaSASLPassword   string
	RedisURL            string
	Environment         string
	LogLevel            string
	MetricsPath         string
	OpenAPIPath         string
	MigrationsPath      string
	ShutdownTimeout     time.Duration
	ReadinessTimeout    time.Duration
	OTelEndpoint        string
	OTelInsecure        bool
	GatewayServiceToken string
	GatewayJWKSURL      string
	GatewayIssuer       string
	GatewayAudience     string
	GatewaySubject      string
	InventoryJWKSURL    string
	CursorSecret        string
}

func Load(source Source) (Config, error) {
	c := Config{
		Role:                value(source, "ORDERS_ROLE", "api"),
		HTTPAddr:            value(source, "ORDERS_HTTP_ADDR", ":8080"),
		DatabaseURL:         value(source, "ORDERS_DATABASE_URL", ""),
		KafkaBrokers:        value(source, "ORDERS_KAFKA_BROKERS", ""),
		KafkaTLS:            boolean(source, "ORDERS_KAFKA_TLS", false),
		KafkaSASLUsername:   value(source, "ORDERS_KAFKA_SASL_USERNAME", ""),
		KafkaSASLPassword:   value(source, "ORDERS_KAFKA_SASL_PASSWORD", ""),
		RedisURL:            value(source, "ORDERS_REDIS_URL", ""),
		Environment:         value(source, "ORDERS_ENVIRONMENT", "development"),
		LogLevel:            value(source, "ORDERS_LOG_LEVEL", "info"),
		MetricsPath:         value(source, "ORDERS_METRICS_PATH", "/metrics"),
		OpenAPIPath:         value(source, "ORDERS_OPENAPI_PATH", "contracts/openapi/orders.openapi.yaml"),
		MigrationsPath:      value(source, "ORDERS_MIGRATIONS_PATH", "database/orders/migrations"),
		ShutdownTimeout:     duration(source, "ORDERS_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadinessTimeout:    duration(source, "ORDERS_READINESS_TIMEOUT", 2*time.Second),
		OTelEndpoint:        value(source, "ORDERS_OTEL_ENDPOINT", ""),
		OTelInsecure:        boolean(source, "ORDERS_OTEL_INSECURE", false),
		GatewayServiceToken: value(source, "ORDERS_GATEWAY_SERVICE_TOKEN", ""),
		GatewayJWKSURL:      value(source, "ORDERS_GATEWAY_JWKS_URL", "http://identity:8080/.well-known/jwks.json"),
		GatewayIssuer:       value(source, "ORDERS_GATEWAY_ISSUER", "identity.local"),
		GatewayAudience:     value(source, "ORDERS_GATEWAY_AUDIENCE", "platform-internal"),
		GatewaySubject:      value(source, "ORDERS_GATEWAY_SUBJECT", "svc_gateway"),
		InventoryJWKSURL:    value(source, "ORDERS_INVENTORY_JWKS_URL", "http://inventory:8080/.well-known/jwks.json"),
		CursorSecret:        value(source, "ORDERS_CURSOR_SECRET", ""),
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
	if c.Role != "api" && c.Role != "relay" {
		return errors.New("ORDERS_ROLE must be api or relay")
	}
	if c.MetricsPath != "/metrics" {
		return errors.New("ORDERS_METRICS_PATH must be /metrics")
	}
	if c.ShutdownTimeout <= 0 || c.ReadinessTimeout <= 0 {
		return errors.New("Orders timeouts must be positive")
	}
	if c.Environment != "development" {
		if strings.TrimSpace(c.DatabaseURL) == "" {
			return errors.New("Orders database configuration is required outside development")
		}
		if len([]byte(c.CursorSecret)) < 32 {
			return errors.New("ORDERS_CURSOR_SECRET must contain at least 32 bytes outside development")
		}
		if c.Role == "relay" && strings.TrimSpace(c.KafkaBrokers) == "" {
			return errors.New("Orders Kafka configuration is required for the relay")
		}
		if c.Role == "relay" && (!c.KafkaTLS || strings.TrimSpace(c.KafkaSASLUsername) == "" || strings.TrimSpace(c.KafkaSASLPassword) == "") {
			return errors.New("secure authenticated Kafka configuration is required for the relay outside development")
		}
		if strings.TrimSpace(c.OTelEndpoint) == "" || c.OTelInsecure {
			return errors.New("secure Orders OpenTelemetry configuration is required outside development")
		}
		if !tlsPostgres(c.DatabaseURL) {
			return errors.New("ORDERS_DATABASE_URL must require PostgreSQL TLS outside development")
		}
		if c.Role == "api" {
			if err := validateURL(c.GatewayJWKSURL, "ORDERS_GATEWAY_JWKS_URL"); err != nil {
				return err
			}
			if !strings.HasPrefix(strings.ToLower(c.GatewayJWKSURL), "https://") {
				return errors.New("ORDERS_GATEWAY_JWKS_URL must use HTTPS outside development")
			}
			if strings.TrimSpace(c.GatewayIssuer) == "" || strings.TrimSpace(c.GatewayAudience) == "" || strings.TrimSpace(c.GatewaySubject) == "" {
				return errors.New("Orders Gateway JWT settings are required outside development")
			}
			if strings.TrimSpace(c.GatewayServiceToken) != "" {
				return errors.New("ORDERS_GATEWAY_SERVICE_TOKEN is allowed only in development")
			}
		}
		if err := validateURL(c.InventoryJWKSURL, "ORDERS_INVENTORY_JWKS_URL"); err != nil {
			return err
		}
		if !strings.HasPrefix(strings.ToLower(c.InventoryJWKSURL), "https://") {
			return errors.New("ORDERS_INVENTORY_JWKS_URL must use HTTPS outside development")
		}
	}
	return nil
}

func validateURL(raw, name string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%s must be an HTTP or HTTPS URL", name)
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
