package orders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
)

const (
	CreateOperation = "orders.create"
	CancelOperation = "orders.cancel"
	MaxPageSize     = 100
)

type Service struct {
	store OrderStore
	ids   IDGenerator
	clock Clock
}

type CreateInput struct {
	Actor          Actor
	CustomerID     string
	Currency       string
	Items          []domain.Item
	IdempotencyKey string
}

type CancelInput struct {
	Actor          Actor
	OrderID        string
	Reason         string
	IdempotencyKey string
}

type ListInput struct {
	Actor      Actor
	CustomerID string
	Status     string
	Cursor     string
	Limit      int
}

func NewService(store OrderStore, ids IDGenerator, clock Clock) (*Service, error) {
	if store == nil || ids == nil || clock == nil {
		return nil, ErrInvalidDependency
	}
	return &Service{store: store, ids: ids, clock: clock}, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResult, error) {
	if err := authorizeActor(input.Actor); err != nil {
		return CreateResult{}, err
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return CreateResult{}, ErrInvalidInput
	}
	if input.CustomerID != input.Actor.Subject && !input.Actor.HasScope("orders:write:any") {
		return CreateResult{}, ErrForbidden
	}
	now := s.clock.Now().UTC()
	order, err := domain.NewOrder(s.ids.NewOrderID(), input.CustomerID, input.Currency, input.Items, now)
	if err != nil {
		return CreateResult{}, errors.Join(ErrInvalidInput, err)
	}
	hash, err := requestHash(struct {
		CustomerID string
		Currency   string
		Items      []domain.Item
	}{input.CustomerID, strings.ToUpper(strings.TrimSpace(input.Currency)), input.Items})
	if err != nil {
		return CreateResult{}, err
	}
	return s.store.Create(ctx, order, CreateOperation, input.IdempotencyKey, hash)
}

func (s *Service) Get(ctx context.Context, actor Actor, orderID string) (domain.Order, error) {
	if err := authorizeActor(actor); err != nil {
		return domain.Order{}, err
	}
	order, err := s.store.FindByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	if order.CustomerID != actor.Subject && !actor.HasScope("orders:read:any") {
		return domain.Order{}, ErrNotFound
	}
	return order, nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]domain.Order, string, error) {
	if err := authorizeActor(input.Actor); err != nil {
		return nil, "", err
	}
	if input.Limit < 1 || input.Limit > MaxPageSize {
		return nil, "", ErrInvalidInput
	}
	target := input.CustomerID
	if target == "" {
		target = input.Actor.Subject
	}
	if target != input.Actor.Subject && !input.Actor.HasScope("orders:read:any") {
		return nil, "", ErrForbidden
	}
	return s.store.List(ctx, target, input.Status, input.Cursor, input.Limit)
}

func (s *Service) Cancel(ctx context.Context, input CancelInput) (domain.Order, error) {
	if err := authorizeActor(input.Actor); err != nil {
		return domain.Order{}, err
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return domain.Order{}, ErrInvalidInput
	}
	order, err := s.store.FindByID(ctx, input.OrderID)
	if err != nil {
		return domain.Order{}, err
	}
	if order.CustomerID != input.Actor.Subject && !input.Actor.HasScope("orders:cancel:any") {
		return domain.Order{}, ErrNotFound
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "CUSTOMER_REQUESTED"
	}
	return s.store.RequestCancellation(ctx, order.ID, reason, order.Version, s.clock.Now().UTC())
}

func authorizeActor(actor Actor) error {
	if !strings.HasPrefix(actor.Subject, "usr_") {
		return ErrUnauthorized
	}
	return nil
}

func requestHash(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
