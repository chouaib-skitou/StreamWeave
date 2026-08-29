package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

type Clock struct{}

func (Clock) Now() time.Time { return time.Now().UTC() }

type SimulatedMailer struct {
	logger   *slog.Logger
	mu       *sync.RWMutex
	messages map[string]mailMessage
}

type mailMessage struct {
	kind      string
	recipient string
	token     string
}

func NewSimulatedMailer(logger *slog.Logger) *SimulatedMailer {
	return &SimulatedMailer{logger: logger, mu: &sync.RWMutex{}, messages: make(map[string]mailMessage)}
}

func (m *SimulatedMailer) Send(ctx context.Context, kind, recipient, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(recipient) == "" || strings.TrimSpace(token) == "" {
		return errors.New("mailer recipient and token are required")
	}
	m.mu.Lock()
	m.messages[messageKey(kind, recipient)] = mailMessage{kind: kind, recipient: recipient, token: token}
	m.mu.Unlock()
	if m.logger != nil {
		m.logger.Info("simulated identity mail queued", "kind", kind)
	}
	return nil
}

func (m *SimulatedMailer) Latest(ctx context.Context, kind, recipient string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.RLock()
	message, ok := m.messages[messageKey(kind, recipient)]
	m.mu.RUnlock()
	if !ok {
		return "", errors.New("mail not found")
	}
	return message.token, nil
}

func messageKey(kind, recipient string) string {
	return strings.ToLower(strings.TrimSpace(kind)) + "\x00" + strings.ToLower(strings.TrimSpace(recipient))
}

type SMTPMailer struct {
	host             string
	port             int
	username         string
	password         string
	from             string
	publicAppURL     string
	verificationPath string
	resetPath        string
}

const (
	verificationPath = "/verify-email"
	resetPath        = "/reset-password"
)

func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, username: username, password: password, from: from, publicAppURL: "http://localhost:3000", verificationPath: verificationPath, resetPath: resetPath}
}

func NewConfiguredSMTPMailer(host string, port int, username, password, from, publicAppURL string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, username: username, password: password, from: from, publicAppURL: publicAppURL, verificationPath: verificationPath, resetPath: resetPath}
}

func (m *SMTPMailer) Send(ctx context.Context, kind, recipient, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	content, err := renderIdentityEmail(kind, recipient, token, m.publicAppURL, m.verificationPath, m.resetPath)
	if err != nil {
		return err
	}
	boundary := "identity-mail-boundary"
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n--%s\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n--%s\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n--%s--\r\n", m.from, recipient, content.Subject, boundary, boundary, content.Text, boundary, content.HTML, boundary)
	address := fmt.Sprintf("%s:%d", m.host, m.port)
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	return smtp.SendMail(address, auth, m.from, []string{recipient}, []byte(body))
}
