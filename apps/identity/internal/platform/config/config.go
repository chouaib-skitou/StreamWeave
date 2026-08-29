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

type Source interface {
	Lookup(key string) (string, bool)
}

type OSEnv struct{}

func (OSEnv) Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

type Config struct {
	HTTPAddr              string
	DatabaseURL           string
	MigrationDir          string
	RunMigrations         bool
	Environment           string
	LogLevel              string
	ShutdownTimeout       time.Duration
	ReadinessTimeout      time.Duration
	MetricsPath           string
	DemoRegistration      bool
	EmergencyRevocation   bool
	Issuer                string
	HumanAudience         string
	MachineAudience       string
	SigningKeyPath        string
	SigningKeyID          string
	RedisURL              string
	KafkaBrokers          string
	OTelEndpoint          string
	MailerMode            string
	TestMailerEnabled     bool
	SMTPHost              string
	SMTPPort              int
	SMTPUsername          string
	SMTPPassword          string
	SMTPFrom              string
	PublicAppURL          string
	EmailVerificationPath string
	PasswordResetPath     string
	RateLoginFailures     int
	RateRefreshMinute     int
	RateResetHour         int
	RateServiceMinute     int
}

func Load(source Source) (Config, error) {
	c := Config{
		HTTPAddr:              value(source, "IDENTITY_HTTP_ADDR", ":8080"),
		DatabaseURL:           value(source, "IDENTITY_DATABASE_URL", ""),
		MigrationDir:          value(source, "IDENTITY_MIGRATION_DIR", "database/identity/migrations"),
		Environment:           value(source, "IDENTITY_ENVIRONMENT", "development"),
		LogLevel:              value(source, "IDENTITY_LOG_LEVEL", "info"),
		MetricsPath:           value(source, "IDENTITY_METRICS_PATH", "/metrics"),
		ShutdownTimeout:       duration(source, "IDENTITY_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadinessTimeout:      duration(source, "IDENTITY_READINESS_TIMEOUT", 2*time.Second),
		RunMigrations:         boolean(source, "IDENTITY_RUN_MIGRATIONS", true),
		DemoRegistration:      boolean(source, "IDENTITY_DEMO_REGISTRATION_ENABLED", false),
		EmergencyRevocation:   boolean(source, "IDENTITY_EMERGENCY_REVOCATION_ENABLED", true),
		Issuer:                value(source, "IDENTITY_ISSUER", "identity.local"),
		HumanAudience:         value(source, "IDENTITY_HUMAN_AUDIENCE", "platform-api"),
		MachineAudience:       value(source, "IDENTITY_MACHINE_AUDIENCE", "platform-internal"),
		SigningKeyPath:        value(source, "IDENTITY_SIGNING_KEY_PATH", ""),
		SigningKeyID:          value(source, "IDENTITY_SIGNING_KEY_ID", "identity-local"),
		RedisURL:              value(source, "IDENTITY_REDIS_URL", "redis://localhost:6379/0"),
		KafkaBrokers:          value(source, "IDENTITY_KAFKA_BROKERS", "localhost:9092"),
		OTelEndpoint:          value(source, "IDENTITY_OTEL_ENDPOINT", ""),
		MailerMode:            value(source, "IDENTITY_MAILER_MODE", "simulated"),
		TestMailerEnabled:     boolean(source, "IDENTITY_TEST_MAILER_ENABLED", false),
		SMTPHost:              value(source, "IDENTITY_SMTP_HOST", ""),
		SMTPPort:              integer(source, "IDENTITY_SMTP_PORT", 1025),
		SMTPUsername:          value(source, "IDENTITY_SMTP_USERNAME", ""),
		SMTPPassword:          value(source, "IDENTITY_SMTP_PASSWORD", ""),
		SMTPFrom:              value(source, "IDENTITY_SMTP_FROM", "identity@localhost"),
		PublicAppURL:          value(source, "IDENTITY_PUBLIC_APP_URL", "http://localhost:3000"),
		EmailVerificationPath: value(source, "IDENTITY_EMAIL_VERIFICATION_PATH", "/verify-email"),
		PasswordResetPath:     value(source, "IDENTITY_PASSWORD_RESET_PATH", "/reset-password"),
		RateLoginFailures:     integer(source, "IDENTITY_RATE_LOGIN_FAILURES", 5),
		RateRefreshMinute:     integer(source, "IDENTITY_RATE_REFRESH_PER_MINUTE", 30),
		RateResetHour:         integer(source, "IDENTITY_RATE_RESET_PER_HOUR", 3),
		RateServiceMinute:     integer(source, "IDENTITY_RATE_SERVICE_TOKEN_PER_MINUTE", 30),
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return errors.New("IDENTITY_HTTP_ADDR must not be empty")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("IDENTITY_DATABASE_URL must not be empty")
	}
	databaseURL, err := url.Parse(c.DatabaseURL)
	if err != nil || databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql" {
		return errors.New("IDENTITY_DATABASE_URL must be a PostgreSQL URL")
	}
	if strings.TrimSpace(c.MigrationDir) == "" {
		return errors.New("IDENTITY_MIGRATION_DIR must not be empty")
	}
	if c.ShutdownTimeout <= 0 || c.ReadinessTimeout <= 0 {
		return errors.New("Identity timeouts must be positive")
	}
	if c.MetricsPath == "" || !strings.HasPrefix(c.MetricsPath, "/") {
		return errors.New("IDENTITY_METRICS_PATH must be an absolute HTTP path")
	}
	if c.Environment == "production" && c.DemoRegistration {
		return errors.New("demo registration must be disabled in production")
	}
	if c.Environment == "production" && strings.TrimSpace(c.SigningKeyPath) == "" {
		return errors.New("IDENTITY_SIGNING_KEY_PATH is required in production")
	}
	if c.MailerMode != "simulated" && c.MailerMode != "smtp" {
		return errors.New("IDENTITY_MAILER_MODE must be simulated or smtp")
	}
	if c.TestMailerEnabled && c.MailerMode != "simulated" {
		return errors.New("IDENTITY_TEST_MAILER_ENABLED requires simulated mailer mode")
	}
	if c.Environment == "production" && c.MailerMode != "smtp" {
		return errors.New("IDENTITY_MAILER_MODE must be smtp in production")
	}
	if c.MailerMode == "smtp" && (strings.TrimSpace(c.SMTPHost) == "" || c.SMTPPort <= 0 || strings.TrimSpace(c.SMTPFrom) == "") {
		return errors.New("SMTP host, port, and from are required when SMTP mailer is enabled")
	}
	if c.Environment == "production" && (strings.TrimSpace(c.SMTPUsername) == "" || strings.TrimSpace(c.SMTPPassword) == "") {
		return errors.New("SMTP username and password are required in production")
	}
	publicAppURL, err := url.Parse(c.PublicAppURL)
	if err != nil || (publicAppURL.Scheme != "http" && publicAppURL.Scheme != "https") || strings.TrimSpace(publicAppURL.Host) == "" {
		return errors.New("IDENTITY_PUBLIC_APP_URL must be an HTTP or HTTPS URL")
	}
	if !strings.HasPrefix(c.EmailVerificationPath, "/") || !strings.HasPrefix(c.PasswordResetPath, "/") {
		return errors.New("Identity email paths must be absolute HTTP paths")
	}
	if strings.TrimSpace(c.Issuer) == "" || strings.TrimSpace(c.HumanAudience) == "" || strings.TrimSpace(c.MachineAudience) == "" || strings.TrimSpace(c.SigningKeyID) == "" {
		return errors.New("Identity issuer, audiences, and signing key ID are required")
	}
	if c.RateLoginFailures <= 0 || c.RateRefreshMinute <= 0 || c.RateResetHour <= 0 || c.RateServiceMinute <= 0 {
		return errors.New("Identity rate limits must be positive")
	}
	return nil
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
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return -1
	}
	return parsed
}

func boolean(source Source, key string, fallback bool) bool {
	v, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return parsed
}

func integer(source Source, key string, fallback int) int {
	v, ok := source.Lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return -1
	}
	return parsed
}

func (c Config) SafeSummary() string {
	return fmt.Sprintf("environment=%s http_addr=%s migrations=%t", c.Environment, c.HTTPAddr, c.RunMigrations)
}
