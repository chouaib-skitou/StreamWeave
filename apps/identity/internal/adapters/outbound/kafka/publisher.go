package kafka

import (
	"context"
	"strings"
	"time"

	identitypostgres "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/postgres"
	segmentkafka "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Publisher struct{ writer *segmentkafka.Writer }

func NewPublisher(brokers string) *Publisher {
	addresses := make([]string, 0)
	for _, address := range strings.Split(brokers, ",") {
		if trimmed := strings.TrimSpace(address); trimmed != "" {
			addresses = append(addresses, trimmed)
		}
	}
	return &Publisher{writer: &segmentkafka.Writer{Addr: segmentkafka.TCP(addresses...), Topic: "commerce.security.audit.v1", Balancer: &segmentkafka.Hash{}, RequiredAcks: segmentkafka.RequireAll, Async: false}}
}

func (p *Publisher) Publish(ctx context.Context, event identitypostgres.OutboxEvent) error {
	ctx, span := otel.Tracer("identity/outbox").Start(ctx, "outbox publish")
	defer span.End()
	span.SetAttributes(attribute.String("messaging.system", "kafka"), attribute.String("messaging.operation.type", "publish"), attribute.String("identity.event_type", event.EventType))
	return p.writer.WriteMessages(ctx, segmentkafka.Message{Key: []byte(event.AggregateID), Value: event.Payload, Headers: []segmentkafka.Header{{Key: "event_type", Value: []byte(event.EventType)}, {Key: "event_id", Value: []byte(event.ID.String())}}})
}
func (p *Publisher) Close() error { return p.writer.Close() }

type Relay struct {
	store           *identitypostgres.Store
	publisher       *Publisher
	publishObserver func(string)
}

func NewRelay(store *identitypostgres.Store, publisher *Publisher) *Relay {
	return &Relay{store: store, publisher: publisher}
}
func (r *Relay) SetPublishObserver(observer func(string)) { r.publishObserver = observer }
func (r *Relay) Run(ctx context.Context) error {
	ticker := timeTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := r.publishBatch(ctx); err != nil && ctx.Err() == nil { /* retry on next poll; durable failure remains in outbox */
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (r *Relay) publishBatch(ctx context.Context) error {
	events, err := r.store.PendingOutbox(ctx, 100)
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := r.publisher.Publish(ctx, event); err != nil {
			r.observePublish("error")
			_ = r.store.MarkOutboxFailed(ctx, event.ID, boundedError(err))
			return err
		}
		if err := r.store.MarkOutboxPublished(ctx, event.ID, time.Now().UTC()); err != nil {
			r.observePublish("error")
			return err
		}
		r.observePublish("success")
	}
	return nil
}
func (r *Relay) observePublish(outcome string) {
	if r.publishObserver != nil {
		r.publishObserver(outcome)
	}
}
func boundedError(err error) string {
	if err == nil {
		return ""
	}
	value := err.Error()
	if len(value) > 256 {
		return value[:256]
	}
	return value
}

var timeTicker = func(interval time.Duration) *time.Ticker { return time.NewTicker(interval) }
