package orders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
)

const (
	CreateOperation = "orders.create"
	CancelOperation = "orders.cancel"
	MaxPageSize     = 100
)

var (
	idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{16,256}$`)
	reasonPattern         = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,31}$`)
	actorPattern          = regexp.MustCompile(`^usr_[A-Za-z0-9]+$`)
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
	if !idempotencyKeyPattern.MatchString(input.IdempotencyKey) {
		return CreateResult{}, ErrInvalidInput
	}
	if input.CustomerID != input.Actor.Subject && !input.Actor.HasScope("orders:write:any") {
		return CreateResult{}, ErrForbidden
	}
	if input.CustomerID == input.Actor.Subject && !input.Actor.HasScope("orders:write:self") && !input.Actor.HasScope("orders:write:any") {
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
	}{input.CustomerID, strings.ToUpper(strings.TrimSpace(input.Currency)), order.Items})
	if err != nil {
		return CreateResult{}, err
	}
	return s.store.Create(ctx, input.Actor.Subject, order, CreateOperation, input.IdempotencyKey, hash)
}

func (s *Service) Get(ctx context.Context, actor Actor, orderID string) (domain.Order, error) {
	if err := authorizeActor(actor); err != nil {
		return domain.Order{}, err
	}
	if !domain.ValidOrderID(orderID) {
		return domain.Order{}, ErrInvalidInput
	}
	order, err := s.store.FindByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	if order.CustomerID != actor.Subject && !actor.HasScope("orders:read:any") {
		return domain.Order{}, ErrNotFound
	}
	if order.CustomerID == actor.Subject && !actor.HasScope("orders:read:self") && !actor.HasScope("orders:read:any") {
		return domain.Order{}, ErrForbidden
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
	if len(input.Cursor) > 256 {
		return nil, "", ErrInvalidCursor
	}
	if input.Status != "" && !validStatus(input.Status) {
		return nil, "", ErrInvalidInput
	}
	target := input.CustomerID
	if target == "" {
		target = input.Actor.Subject
	}
	if target != input.Actor.Subject && !input.Actor.HasScope("orders:read:any") {
		return nil, "", ErrForbidden
	}
	if target == input.Actor.Subject && !input.Actor.HasScope("orders:read:self") && !input.Actor.HasScope("orders:read:any") {
		return nil, "", ErrForbidden
	}
	if !domain.ValidCustomerID(target) {
		return nil, "", ErrInvalidInput
	}
	return s.store.List(ctx, target, input.Status, input.Cursor, input.Limit)
}

func (s *Service) Cancel(ctx context.Context, input CancelInput) (domain.Order, error) {
	if err := authorizeActor(input.Actor); err != nil {
		return domain.Order{}, err
	}
	if !idempotencyKeyPattern.MatchString(input.IdempotencyKey) {
		return domain.Order{}, ErrInvalidInput
	}
	if !domain.ValidOrderID(input.OrderID) {
		return domain.Order{}, ErrInvalidInput
	}
	order, err := s.store.FindByID(ctx, input.OrderID)
	if err != nil {
		return domain.Order{}, err
	}
	if order.CustomerID != input.Actor.Subject && !input.Actor.HasScope("orders:cancel:any") {
		return domain.Order{}, ErrNotFound
	}
	if order.CustomerID == input.Actor.Subject && !input.Actor.HasScope("orders:cancel:self") && !input.Actor.HasScope("orders:cancel:any") {
		return domain.Order{}, ErrForbidden
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "CUSTOMER_REQUESTED"
	}
	if !reasonPattern.MatchString(reason) {
		return domain.Order{}, ErrInvalidInput
	}
	hash, err := requestHash(struct{ OrderID, Reason string }{input.OrderID, reason})
	if err != nil {
		return domain.Order{}, err
	}
	return s.store.RequestCancellation(ctx, input.Actor.Subject, order.ID, CancelOperation, input.IdempotencyKey, hash, reason, order.Version, s.clock.Now().UTC())
}

func authorizeActor(actor Actor) error {
	if actor.Type != "human" || !actorPattern.MatchString(actor.Subject) {
		return ErrUnauthorized
	}
	return nil
}

func validStatus(value string) bool {
	switch value {
	case string(domain.StatusPending), string(domain.StatusConfirmed), string(domain.StatusCancelReq), string(domain.StatusCancelled), string(domain.StatusCompleted), string(domain.StatusFailed):
		return true
	default:
		return false
	}
}

func requestHash(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
