package orders

import (
	"context"
	"time"

	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
)

type Clock interface{ Now() time.Time }

type IDGenerator interface{ NewOrderID() string }

type OrderStore interface {
	Create(ctx context.Context, order domain.Order, idempotencyKey string) error
	FindByID(ctx context.Context, orderID string) (domain.Order, error)
	ListByCustomer(ctx context.Context, customerID, cursor string, limit int) ([]domain.Order, string, error)
	Transition(ctx context.Context, orderID string, expectedVersion uint64, status domain.Status, at time.Time) error
}

type EventPublisher interface {
	Publish(ctx context.Context, order domain.Order, eventType string) error
}
