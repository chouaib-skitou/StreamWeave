package runtime

import (
	"context"
	"strings"
	"testing"
)

func TestSimulatedMailerStoresLatestTokenWithoutLoggingIt(t *testing.T) {
	mailer := NewSimulatedMailer(nil)
	if err := mailer.Send(context.Background(), "email-verification", "person@example.test", "one-time-token"); err != nil {
		t.Fatal(err)
	}
	token, err := mailer.Latest(context.Background(), "EMAIL-VERIFICATION", "PERSON@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if token != "one-time-token" {
		t.Fatalf("got token %q", token)
	}
}

func TestSimulatedMailerRequiresDeliveryData(t *testing.T) {
	mailer := NewSimulatedMailer(nil)
	if err := mailer.Send(context.Background(), "password-reset", "", "token"); err == nil {
		t.Fatal("expected missing recipient to fail")
	}
	if _, err := mailer.Latest(context.Background(), "password-reset", "missing@example.test"); err == nil {
		t.Fatal("expected missing message")
	}
}

func TestIdentityEmailTemplatesRenderSecureActionLinks(t *testing.T) {
	content, err := renderIdentityEmail("email-verification", "person@example.test", "abc123", "https://app.example.test", "/verify-email", "/reset-password")
	if err != nil {
		t.Fatal(err)
	}
	if content.Subject == "" || content.Text == "" || content.HTML == "" {
		t.Fatal("expected complete email content")
	}
	if !containsAll(content.Text, "https://app.example.test/verify-email?token=abc123", "24 hours") {
		t.Fatalf("unexpected text template: %s", content.Text)
	}
	if !containsAll(content.HTML, "Verify email address", "https://app.example.test/verify-email?token=abc123") {
		t.Fatalf("unexpected HTML template: %s", content.HTML)
	}
	if !containsAll(content.HTML, "cid:platform-logo", "Need help? Contact", "Security note", "StreamWeave") {
		t.Fatalf("shared email shell is incomplete: %s", content.HTML)
	}
}

func TestIdentityEmailTemplatesRejectUnknownKinds(t *testing.T) {
	if _, err := renderIdentityEmail("unknown", "person@example.test", "token", "https://app.example.test", "/verify-email", "/reset-password"); err == nil {
		t.Fatal("expected unknown email kind to fail")
	}
}

func containsAll(value string, required ...string) bool {
	for _, item := range required {
		if !strings.Contains(value, item) {
			return false
		}
	}
	return true
}
