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
	HTTPAddr            string
	Environment         string
	LogLevel            string
	MetricsPath         string
	OpenAPIPath         string
	ShutdownTimeout     time.Duration
	ReadinessTimeout    time.Duration
	IdentityURL         string
	OrdersURL           string
	JWKSURL             string
	Issuer              string
	HumanAudience       string
	RedisURL            string
	ServiceClientID     string
	ServiceClientSecret string
	ServiceScopes       []string
	StaticServiceToken  string
	TrustProxyHeaders   bool
	MaxResponseBytes    int64
	OTelEndpoint        string
	OTelInsecure        bool
	CORSOrigins         []string
}

func Load(source Source) (Config, error) {
	c := Config{
		HTTPAddr:            value(source, "GATEWAY_HTTP_ADDR", ":8080"),
		Environment:         value(source, "GATEWAY_ENVIRONMENT", "development"),
		LogLevel:            value(source, "GATEWAY_LOG_LEVEL", "info"),
		MetricsPath:         value(source, "GATEWAY_METRICS_PATH", "/metrics"),
		OpenAPIPath:         value(source, "GATEWAY_OPENAPI_PATH", "contracts/openapi/gateway.openapi.yaml"),
		ShutdownTimeout:     duration(source, "GATEWAY_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadinessTimeout:    duration(source, "GATEWAY_READINESS_TIMEOUT", 2*time.Second),
		IdentityURL:         value(source, "GATEWAY_IDENTITY_URL", "http://localhost:8080"),
		OrdersURL:           value(source, "GATEWAY_ORDERS_URL", "http://localhost:8082"),
		JWKSURL:             value(source, "GATEWAY_JWKS_URL", "http://localhost:8080/.well-known/jwks.json"),
		Issuer:              value(source, "GATEWAY_ISSUER", "identity.local"),
		HumanAudience:       value(source, "GATEWAY_HUMAN_AUDIENCE", "platform-api"),
		RedisURL:            value(source, "GATEWAY_REDIS_URL", "redis://localhost:6379/1"),
		ServiceClientID:     value(source, "GATEWAY_SERVICE_CLIENT_ID", ""),
		ServiceClientSecret: value(source, "GATEWAY_SERVICE_CLIENT_SECRET", ""),
		ServiceScopes:       csv(source, "GATEWAY_SERVICE_SCOPES", []string{"orders:read:self", "orders:read:any", "orders:write:self", "orders:cancel", "identity:sessions:read", "identity:sessions:write"}),
		StaticServiceToken:  value(source, "GATEWAY_SERVICE_TOKEN", ""),
		TrustProxyHeaders:   boolean(source, "GATEWAY_TRUST_PROXY_HEADERS", false),
		MaxResponseBytes:    int64(integer(source, "GATEWAY_MAX_RESPONSE_BYTES", 4<<20)),
		OTelEndpoint:        value(source, "GATEWAY_OTEL_ENDPOINT", ""),
		OTelInsecure:        boolean(source, "GATEWAY_OTEL_INSECURE", false),
		CORSOrigins:         csv(source, "GATEWAY_CORS_ORIGINS", nil),
	}
	for _, key := range []string{"GATEWAY_TRUST_PROXY_HEADERS", "GATEWAY_OTEL_INSECURE"} {
		if err := validateBoolean(source, key); err != nil {
			return Config{}, err
		}
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return errors.New("GATEWAY_HTTP_ADDR must not be empty")
	}
	if c.Environment != "development" && c.Environment != "staging" && c.Environment != "production" {
		return errors.New("GATEWAY_ENVIRONMENT must be development, staging, or production")
	}
	if c.ShutdownTimeout <= 0 || c.ReadinessTimeout <= 0 {
		return errors.New("Gateway timeouts must be positive")
	}
	if c.MetricsPath == "" || !strings.HasPrefix(c.MetricsPath, "/") {
		return errors.New("GATEWAY_METRICS_PATH must be an absolute HTTP path")
	}
	if err := validateURL(c.IdentityURL, "GATEWAY_IDENTITY_URL"); err != nil {
		return err
	}
	if err := validateURL(c.OrdersURL, "GATEWAY_ORDERS_URL"); err != nil {
		return err
	}
	if err := validateURL(c.JWKSURL, "GATEWAY_JWKS_URL"); err != nil {
		return err
	}
	if strings.TrimSpace(c.Issuer) == "" || strings.TrimSpace(c.HumanAudience) == "" {
		return errors.New("Gateway issuer and human audience are required")
	}
	if c.MaxResponseBytes <= 0 || c.MaxResponseBytes > 16<<20 {
		return errors.New("GATEWAY_MAX_RESPONSE_BYTES must be between 1 and 16777216")
	}
	if len(c.ServiceScopes) == 0 {
		return errors.New("GATEWAY_SERVICE_SCOPES must not be empty")
	}
	if c.Environment != "development" {
		if strings.TrimSpace(c.ServiceClientID) == "" || strings.TrimSpace(c.ServiceClientSecret) == "" {
			return errors.New("service client credentials are required outside development")
		}
		if c.StaticServiceToken != "" {
			return errors.New("static service token is allowed only in development")
		}
	}
	if c.Environment == "production" {
		if !isHTTPS(c.IdentityURL) || !isHTTPS(c.OrdersURL) || !isHTTPS(c.JWKSURL) {
			return errors.New("internal HTTP and JWKS URLs must use HTTPS in production")
		}
		if !isRediss(c.RedisURL) {
			return errors.New("GATEWAY_REDIS_URL must use rediss:// in production")
		}
	}
	if c.Environment == "staging" && (!isHTTPS(c.IdentityURL) || !isHTTPS(c.OrdersURL) || !isHTTPS(c.JWKSURL) || !isRediss(c.RedisURL)) {
		return errors.New("staging internal HTTP and Redis URLs must use TLS")
	}
	if c.Environment != "development" && c.StaticServiceToken != "" {
		return errors.New("static service token is allowed only in development")
	}
	if len(c.CORSOrigins) > 0 && contains(c.CORSOrigins, "*") {
		return errors.New("wildcard CORS origin is not allowed")
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
func isHTTPS(raw string) bool  { return strings.HasPrefix(strings.ToLower(raw), "https://") }
func isRediss(raw string) bool { return strings.HasPrefix(strings.ToLower(raw), "rediss://") }
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}
func value(source Source, key, fallback string) string {
	if value, ok := source.Lookup(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
func csv(source Source, key string, fallback []string) []string {
	raw, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	values := strings.Split(raw, ",")
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}
func duration(source Source, key string, fallback time.Duration) time.Duration {
	raw, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	result, err := time.ParseDuration(raw)
	if err != nil {
		return -1
	}
	return result
}
func integer(source Source, key string, fallback int) int {
	raw, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	result, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return result
}
func boolean(source Source, key string, fallback bool) bool {
	raw, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	result, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return result
}
func validateBoolean(source Source, key string) error {
	raw, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	if _, err := strconv.ParseBool(raw); err != nil {
		return fmt.Errorf("%s must be a boolean", key)
	}
	return nil
}
