package runtime

import (
	"context"
	"log/slog"
	"time"
)

type Clock struct{}

func (Clock) Now() time.Time { return time.Now().UTC() }

type SimulatedMailer struct{ logger *slog.Logger }

func NewSimulatedMailer(logger *slog.Logger) SimulatedMailer { return SimulatedMailer{logger: logger} }
func (m SimulatedMailer) Send(_ context.Context, kind, token string) error {
	if m.logger != nil {
		m.logger.Info("simulated identity mail queued", "kind", kind)
	}
	return nil
}
