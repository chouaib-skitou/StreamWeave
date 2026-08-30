package config

import (
	"testing"
	"time"
)

type mapSource map[string]string

func (m mapSource) Lookup(key string) (string, bool) {
	v, ok := m[key]
	return v, ok
}

func TestLoadAppliesSafeDefaults(t *testing.T) {
	cfg, err := Load(mapSource{"IDENTITY_DATABASE_URL": "postgres://identity:secret@localhost/identity"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" || cfg.MetricsPath != "/metrics" || cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsProductionDemoRegistration(t *testing.T) {
	_, err := Load(mapSource{
		"IDENTITY_DATABASE_URL":              "postgres://identity:secret@localhost/identity",
		"IDENTITY_ENVIRONMENT":               "production",
		"IDENTITY_DEMO_REGISTRATION_ENABLED": "true",
	})
	if err == nil {
		t.Fatal("expected production demo registration to be rejected")
	}
}

func TestLoadRejectsMalformedDuration(t *testing.T) {
	_, err := Load(mapSource{
		"IDENTITY_DATABASE_URL":     "postgres://identity:secret@localhost/identity",
		"IDENTITY_SHUTDOWN_TIMEOUT": "invalid",
	})
	if err == nil {
		t.Fatal("expected malformed duration to be rejected")
	}
}

func TestLoadRejectsMalformedBoolean(t *testing.T) {
	_, err := Load(mapSource{
		"IDENTITY_DATABASE_URL":    "postgres://identity:secret@localhost/identity",
		"IDENTITY_MIGRATIONS_ONLY": "sometimes",
	})
	if err == nil {
		t.Fatal("expected malformed boolean to be rejected")
	}
}

func TestLoadRejectsMigrationsOnlyWithoutMigrations(t *testing.T) {
	_, err := Load(mapSource{
		"IDENTITY_DATABASE_URL":    "postgres://identity:secret@localhost/identity",
		"IDENTITY_MIGRATIONS_ONLY": "true",
		"IDENTITY_RUN_MIGRATIONS":  "false",
	})
	if err == nil {
		t.Fatal("expected migrations-only mode without migrations to be rejected")
	}
}

func TestValidateRejectsNonPostgresURL(t *testing.T) {
	cfg := Config{HTTPAddr: ":8080", DatabaseURL: "redis://localhost", MigrationDir: "migrations", ShutdownTimeout: time.Second, ReadinessTimeout: time.Second, MetricsPath: "/metrics"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected non-PostgreSQL URL to be rejected")
	}
}

func TestLoadAcceptsSMTPMailerConfiguration(t *testing.T) {
	cfg, err := Load(mapSource{
		"IDENTITY_DATABASE_URL":     "postgres://identity:secret@localhost/identity",
		"IDENTITY_ENVIRONMENT":      "production",
		"IDENTITY_SIGNING_KEY_PATH": "/var/run/secrets/identity/signing-key.pem",
		"IDENTITY_MAILER_MODE":      "smtp",
		"IDENTITY_SMTP_HOST":        "smtp.example.test",
		"IDENTITY_SMTP_PORT":        "587",
		"IDENTITY_SMTP_USERNAME":    "api",
		"IDENTITY_SMTP_PASSWORD":    "secret",
		"IDENTITY_SMTP_FROM":        "identity@example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MailerMode != "smtp" || cfg.SMTPPort != 587 {
		t.Fatalf("unexpected SMTP configuration: %+v", cfg)
	}
}

func TestValidateRejectsProductionSimulatedMailer(t *testing.T) {
	_, err := Load(mapSource{
		"IDENTITY_DATABASE_URL":     "postgres://identity:secret@localhost/identity",
		"IDENTITY_ENVIRONMENT":      "production",
		"IDENTITY_SIGNING_KEY_PATH": "/var/run/secrets/identity/signing-key.pem",
	})
	if err == nil {
		t.Fatal("expected production simulated mailer to be rejected")
	}
}
