package orders

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
)

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeIDs struct{}

func (fakeIDs) NewOrderID() string { return "ord_generated" }

type fakeStore struct {
	created        domain.Order
	result         CreateResult
	err            error
	listedCustomer string
	cancelled      string
}

func (s *fakeStore) Create(_ context.Context, _ string, order domain.Order, _, _, _ string) (CreateResult, error) {
	s.created = order
	if s.err != nil {
		return CreateResult{}, s.err
	}
	if s.result.Order.ID != "" {
		return s.result, nil
	}
	return CreateResult{Order: order}, nil
}
func (s *fakeStore) FindByID(context.Context, string) (domain.Order, error) {
	if s.err != nil {
		return domain.Order{}, s.err
	}
	return s.created, nil
}
func (s *fakeStore) List(_ context.Context, customerID, _, _ string, _ int) ([]domain.Order, string, error) {
	s.listedCustomer = customerID
	return []domain.Order{s.created}, "next", s.err
}
func (s *fakeStore) RequestCancellation(_ context.Context, _, orderID, _, _, _, _ string, _ uint64, _ time.Time) (domain.Order, error) {
	s.cancelled = orderID
	return s.created, s.err
}

func newService(t *testing.T, store *fakeStore) *Service {
	t.Helper()
	service, err := NewService(store, fakeIDs{}, fakeClock{now: time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return service
}
func actor(scopes ...string) Actor {
	if len(scopes) == 0 {
		scopes = []string{"orders:read:self", "orders:write:self", "orders:cancel:self"}
	}
	return Actor{Subject: "usr_1", Type: "human", Scopes: scopes}
}
func orderForStore(t *testing.T) domain.Order {
	t.Helper()
	order, err := domain.NewOrder("ord_1", "usr_1", "EUR", []domain.Item{{ProductID: "prd_1", Quantity: 1}}, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return order
}

func TestCreateEnforcesOwnershipAndDelegatesIdempotency(t *testing.T) {
	store := &fakeStore{}
	service := newService(t, store)
	result, err := service.Create(context.Background(), CreateInput{Actor: actor(), CustomerID: "usr_1", Currency: "EUR", Items: []domain.Item{{ProductID: "prd_1", Quantity: 1}}, IdempotencyKey: "create-000000000001"})
	if err != nil || result.Order.ID != "ord_generated" || store.created.CustomerID != "usr_1" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := service.Create(context.Background(), CreateInput{Actor: actor(), CustomerID: "usr_2", IdempotencyKey: "create-000000000002"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
	if _, err := service.Create(context.Background(), CreateInput{Actor: actor(), CustomerID: "usr_1"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err=%v", err)
	}
}

func TestReadsAndListsDoNotEnumerateOtherCustomers(t *testing.T) {
	store := &fakeStore{created: orderForStore(t)}
	service := newService(t, store)
	if _, err := service.Get(context.Background(), actor(), "ord_1"); err != nil {
		t.Fatal(err)
	}
	store.created.CustomerID = "usr_2"
	if _, err := service.Get(context.Background(), actor(), "ord_1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	if _, _, err := service.List(context.Background(), ListInput{Actor: actor(), CustomerID: "usr_2", Limit: 25}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
	if _, _, err := service.List(context.Background(), ListInput{Actor: actor(), Limit: 101}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err=%v", err)
	}
}

func TestOperatorCanReadAndCancelAnyOrder(t *testing.T) {
	store := &fakeStore{created: orderForStore(t)}
	store.created.CustomerID = "usr_2"
	service := newService(t, store)
	operator := actor("orders:read:any", "orders:cancel:any")
	if _, err := service.Get(context.Background(), operator, "ord_1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.List(context.Background(), ListInput{Actor: operator, CustomerID: "usr_2", Limit: 25}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Cancel(context.Background(), CancelInput{Actor: operator, OrderID: "ord_1", IdempotencyKey: "cancel-000000000001"}); err != nil {
		t.Fatal(err)
	}
	if store.cancelled != "ord_1" {
		t.Fatal("cancel was not delegated")
	}
}

func TestServiceRequiresSpecificDependenciesAndRejectsUnauthenticatedActor(t *testing.T) {
	if _, err := NewService(nil, fakeIDs{}, fakeClock{}); !errors.Is(err, ErrInvalidDependency) {
		t.Fatalf("err=%v", err)
	}
	service := newService(t, &fakeStore{})
	if _, err := service.Get(context.Background(), Actor{}, "ord_1"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err=%v", err)
	}
}

func TestApplicationPropagatesStoreFailuresAndValidatesCancelInput(t *testing.T) {
	store := &fakeStore{created: orderForStore(t), err: errors.New("database unavailable")}
	service := newService(t, store)
	input := CreateInput{Actor: actor(), CustomerID: "usr_1", Currency: "EUR", Items: []domain.Item{{ProductID: "prd_1", Quantity: 1}}, IdempotencyKey: "create-000000000003"}
	if _, err := service.Create(context.Background(), input); err == nil {
		t.Fatal("create swallowed store failure")
	}
	if _, err := service.Get(context.Background(), actor(), "ord_1"); err == nil {
		t.Fatal("get swallowed store failure")
	}
	if _, _, err := service.List(context.Background(), ListInput{Actor: actor(), Limit: 25}); err == nil {
		t.Fatal("list swallowed store failure")
	}
	if _, err := service.Cancel(context.Background(), CancelInput{Actor: actor(), OrderID: "ord_1"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("cancel err=%v", err)
	}
	if _, err := service.Cancel(context.Background(), CancelInput{Actor: actor(), OrderID: "ord_1", IdempotencyKey: "cancel-000000000002"}); err == nil {
		t.Fatal("cancel swallowed store failure")
	}
}

func TestWriteOperatorCanCreateForAnotherCustomer(t *testing.T) {
	store := &fakeStore{}
	service := newService(t, store)
	result, err := service.Create(context.Background(), CreateInput{Actor: actor("orders:write:any"), CustomerID: "usr_2", Currency: "EUR", Items: []domain.Item{{ProductID: "prd_1", Quantity: 1}}, IdempotencyKey: "create-000000000004"})
	if err != nil || result.Order.CustomerID != "usr_2" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestServiceRejectsInvalidActorAndFilters(t *testing.T) {
	service := newService(t, &fakeStore{created: orderForStore(t)})
	if _, err := service.Create(context.Background(), CreateInput{Actor: Actor{Subject: "usr_1", Type: "machine", Scopes: []string{"orders:write:self"}}, CustomerID: "usr_1", Currency: "EUR", Items: []domain.Item{{ProductID: "prd_1", Quantity: 1}}, IdempotencyKey: "create-000000000010"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("actor error=%v", err)
	}
	if _, err := service.Create(context.Background(), CreateInput{Actor: Actor{Subject: "usr_1", Type: "human"}, CustomerID: "usr_1", Currency: "EUR", Items: []domain.Item{{ProductID: "prd_1", Quantity: 1}}, IdempotencyKey: "create-000000000011"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("scope error=%v", err)
	}
	if _, err := service.Get(context.Background(), actor("orders:read:self"), "bad"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("id error=%v", err)
	}
	if _, err := service.Get(context.Background(), actor("orders:write:self"), "ord_1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("read scope error=%v", err)
	}
	if _, _, err := service.List(context.Background(), ListInput{Actor: actor(), Limit: 10, Cursor: strings.Repeat("x", 257)}); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("cursor error=%v", err)
	}
	if _, _, err := service.List(context.Background(), ListInput{Actor: actor(), Limit: 10, Status: "UNKNOWN"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("status error=%v", err)
	}
	if _, _, err := service.List(context.Background(), ListInput{Actor: actor("orders:read:any"), Limit: 10, CustomerID: "bad"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("customer error=%v", err)
	}
}

func TestServiceRejectsInvalidCancellationInputs(t *testing.T) {
	service := newService(t, &fakeStore{created: orderForStore(t)})
	inputs := []CancelInput{
		{Actor: actor(), OrderID: "ord_1", IdempotencyKey: "short"},
		{Actor: actor(), OrderID: "bad", IdempotencyKey: "cancel-000000000001"},
		{Actor: actor(), OrderID: "ord_1", IdempotencyKey: "cancel-000000000001", Reason: "bad reason"},
		{Actor: actor("orders:read:self"), OrderID: "ord_1", IdempotencyKey: "cancel-000000000001"},
	}
	for _, input := range inputs {
		if _, err := service.Cancel(context.Background(), input); !errors.Is(err, ErrInvalidInput) && !errors.Is(err, ErrForbidden) {
			t.Fatalf("error=%v", err)
		}
	}
}

func TestValidStatus(t *testing.T) {
	for _, value := range []string{"PENDING", "CONFIRMED", "CANCEL_REQUESTED", "CANCELLED", "COMPLETED", "FAILED"} {
		if !validStatus(value) {
			t.Fatalf("status %s rejected", value)
		}
	}
	if validStatus("UNKNOWN") {
		t.Fatal("unknown status accepted")
	}
}
