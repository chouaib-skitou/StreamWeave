package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	postgres "github.com/chouaib-skitou/streamweave/apps/orders/internal/adapters/outbound/postgres"
	"github.com/chouaib-skitou/streamweave/apps/orders/internal/platform/config"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

const consumerGroup = "orders-saga-v1"

func main() {
	if err := run(); err != nil {
		slog.Error("orders relay failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(config.OSEnv{})
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.KafkaBrokers) == "" {
		return errors.New("ORDERS_KAFKA_BROKERS is required for relay")
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	database, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store := postgres.NewStoreWithQuoteVerifierAndCursor(database, postgres.NewQuoteVerifier(cfg.InventoryJWKSURL), cfg.CursorSecret)
	dialer := &kafka.Dialer{Timeout: 10 * time.Second}
	transport := &kafka.Transport{DialTimeout: 10 * time.Second}
	if cfg.KafkaTLS {
		dialer.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
		dialer.SASLMechanism = plain.Mechanism{Username: cfg.KafkaSASLUsername, Password: cfg.KafkaSASLPassword}
		transport.TLS = dialer.TLS
		transport.SASL = dialer.SASLMechanism
	}
	relay := &relay{store: store, brokers: splitBrokers(cfg.KafkaBrokers), logger: logger, writers: make(map[string]*kafka.Writer), dialer: dialer, transport: transport, owner: "orders-relay-" + uuid.NewString()}
	for _, topic := range []string{"commerce.inventory.events.v1", "commerce.payment.events.v1", "commerce.fulfillment.events.v1"} {
		go relay.consume(ctx, topic)
	}
	return relay.publish(ctx)
}

type relay struct {
	store     *postgres.Store
	brokers   []string
	logger    *slog.Logger
	writers   map[string]*kafka.Writer
	mu        sync.Mutex
	dialer    *kafka.Dialer
	transport kafka.RoundTripper
	owner     string
}

func (r *relay) publish(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.closeWriters()
			return nil
		case <-ticker.C:
			if err := r.publishBatch(ctx); err != nil {
				r.logger.Error("orders outbox poll failed", "error", err)
			}
		}
	}
}

func (r *relay) publishBatch(ctx context.Context) error {
	rows, err := r.store.ClaimPendingOutbox(ctx, r.owner, 100, time.Now().UTC().Add(2*time.Minute))
	if err != nil {
		return err
	}
	for _, row := range rows {
		writer := r.writer(row.Topic)
		headers := make([]kafka.Header, 0)
		var values map[string]string
		if json.Unmarshal(row.Headers, &values) == nil {
			for key, value := range values {
				headers = append(headers, kafka.Header{Key: key, Value: []byte(value)})
			}
		}
		message := kafka.Message{Key: []byte(row.PartitionKey), Value: row.Payload, Headers: headers}
		if err := writer.WriteMessages(ctx, message); err != nil {
			retryAt := time.Now().UTC().Add(outboxBackoff(row.Attempts))
			if markErr := r.store.MarkOutboxFailed(ctx, row.ID, err.Error(), retryAt); markErr != nil {
				r.logger.Error("orders outbox failure state update failed", "topic", row.Topic, "error", markErr)
			}
			r.logger.Error("orders outbox publish failed", "topic", row.Topic, "attempts", row.Attempts)
			continue
		}
		if err := r.store.MarkOutboxPublished(ctx, row.ID, time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}

func (r *relay) consume(ctx context.Context, topic string) {
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: r.brokers, Dialer: r.dialer, Topic: topic, GroupID: consumerGroup, MinBytes: 1, MaxBytes: 4 << 20, CommitInterval: 0})
	defer reader.Close()
	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			r.logger.Error("orders event read failed", "topic", topic, "error", err)
			time.Sleep(time.Second)
			continue
		}
		if err := r.store.ProcessFact(ctx, consumerGroup, message.Value); err != nil {
			r.logger.Error("orders event processing failed", "topic", topic, "error", err)
			dlq, attempts, recordErr := r.store.RecordFactFailure(ctx, consumerGroup, message.Value, err.Error(), time.Now().UTC().Add(inboxBackoff(1)))
			if recordErr != nil {
				r.logger.Error("orders inbox failure state update failed", "topic", topic, "error", recordErr)
				continue
			}
			if dlq {
				if dlqErr := r.publishDLQ(ctx, topic, message.Value, err.Error()); dlqErr != nil {
					r.logger.Error("orders DLQ publish failed", "topic", topic, "error", dlqErr)
					continue
				}
				if eventID := factEventID(message.Value); eventID != "" {
					if markErr := r.store.MarkFactDeadLetter(ctx, consumerGroup, eventID, err.Error()); markErr != nil {
						r.logger.Error("orders DLQ state update failed", "topic", topic, "error", markErr)
						continue
					}
				}
				if commitErr := reader.CommitMessages(ctx, message); commitErr != nil {
					r.logger.Error("orders DLQ offset commit failed", "topic", topic, "error", commitErr)
				}
			} else if !sleepWithContext(ctx, inboxBackoff(int(attempts))) {
				return
			}
			continue
		}
		if err := reader.CommitMessages(ctx, message); err != nil {
			r.logger.Error("orders event commit failed", "topic", topic, "error", err)
		}
	}
}

func factEventID(payload []byte) string {
	var envelope struct {
		EventID string `json:"event_id"`
	}
	if json.Unmarshal(payload, &envelope) != nil {
		return ""
	}
	return envelope.EventID
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (r *relay) publishDLQ(ctx context.Context, sourceTopic string, payload []byte, reason string) error {
	writer := r.writer("commerce.dlq.v1")
	dlqPayload, err := json.Marshal(map[string]any{
		"source_topic":   sourceTopic,
		"consumer_group": consumerGroup,
		"payload":        json.RawMessage(payload),
		"reason":         reason,
		"quarantined_at": time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	return writer.WriteMessages(ctx, kafka.Message{Key: []byte("orders"), Value: dlqPayload})
}

func outboxBackoff(attempts int32) time.Duration {
	delays := []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute, 15 * time.Minute, 30 * time.Minute}
	if attempts < 0 {
		attempts = 0
	}
	if int(attempts) >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempts]
}

func inboxBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return outboxBackoff(int32(attempt - 1))
}

func (r *relay) writer(topic string) *kafka.Writer {
	r.mu.Lock()
	defer r.mu.Unlock()
	if writer, ok := r.writers[topic]; ok {
		return writer
	}
	writer := &kafka.Writer{Addr: kafka.TCP(r.brokers...), Transport: r.transport, Topic: topic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, BatchTimeout: 10 * time.Millisecond}
	r.writers[topic] = writer
	return writer
}
func (r *relay) closeWriters() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, writer := range r.writers {
		_ = writer.Close()
	}
}
func splitBrokers(raw string) []string {
	values := strings.Split(raw, ",")
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}
