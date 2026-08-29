package runtime

import (
	"context"
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
