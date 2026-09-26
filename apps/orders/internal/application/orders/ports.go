package orders

import (
	"context"
	"time"

	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
)

type Clock interface{ Now() time.Time }

type IDGenerator interface{ NewOrderID() string }

type Actor struct {
	Subject string
	Scopes  []string
}

func (a Actor) HasScope(scope string) bool {
	for _, candidate := range a.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

type CreateResult struct {
	Order    domain.Order
	Replayed bool
}

type OrderStore interface {
	Create(ctx context.Context, order domain.Order, operation, idempotencyKey, requestHash string) (CreateResult, error)
	FindByID(ctx context.Context, orderID string) (domain.Order, error)
	List(ctx context.Context, customerID, status, cursor string, limit int) ([]domain.Order, string, error)
	RequestCancellation(ctx context.Context, orderID, reason string, expectedVersion uint64, at time.Time) (domain.Order, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, order domain.Order, eventType string) error
}
