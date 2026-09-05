package orders

import (
	"errors"
	"testing"
	"time"
)

func validOrder(t *testing.T) Order {
	t.Helper()
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	order, err := NewOrder("ord_01", "usr_01", "eur", []Item{{ProductID: "prd_01", Quantity: 2}}, now)
	if err != nil {
		t.Fatal(err)
	}
	return order
}

func TestNewOrderNormalizesAndCopiesInput(t *testing.T) {
	items := []Item{{ProductID: "prd_01", Quantity: 1}}
	order, err := NewOrder("ord_01", "usr_01", "eur", items, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil || order.Currency != "EUR" || order.Status != StatusPending || order.Version != InitialVersion {
		t.Fatalf("order=%+v err=%v", order, err)
	}
	items[0].Quantity = 99
	if order.Items[0].Quantity != 1 {
		t.Fatal("order retained caller-owned item slice")
	}
}

func TestNewOrderRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		id    string
		user  string
		money string
		items []Item
		want  error
	}{
		{"order id", "bad", "usr_1", "EUR", []Item{{"prd_1", 1}}, ErrInvalidOrderID},
		{"customer id", "ord_1", "customer", "EUR", []Item{{"prd_1", 1}}, ErrInvalidCustomerID},
		{"currency", "ord_1", "usr_1", "EU", []Item{{"prd_1", 1}}, ErrInvalidCurrency},
		{"empty", "ord_1", "usr_1", "EUR", nil, ErrEmptyOrder},
		{"product id", "ord_1", "usr_1", "EUR", []Item{{"product", 1}}, ErrInvalidProductID},
		{"quantity", "ord_1", "usr_1", "EUR", []Item{{"prd_1", 0}}, ErrInvalidQuantity},
		{"duplicate", "ord_1", "usr_1", "EUR", []Item{{"prd_1", 1}, {"prd_1", 2}}, ErrDuplicateProduct},
		{"time", "ord_1", "usr_1", "EUR", []Item{{"prd_1", 1}}, ErrInvalidTimestamp},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			at := now
			if tc.name == "time" {
				at = now.In(time.FixedZone("test", 3600))
			}
			_, err := NewOrder(tc.id, tc.user, tc.money, tc.items, at)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
		})
	}
}

func TestOrderTransitions(t *testing.T) {
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	order := validOrder(t)
	if err := order.Transition(StatusConfirmed, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := order.Transition(StatusCompleted, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := order.Transition(StatusPending, now.Add(3*time.Minute)); !errors.Is(err, ErrTerminalOrder) {
		t.Fatalf("err=%v", err)
	}
}

func TestOrderCancellationAndInvalidTransitions(t *testing.T) {
	order := validOrder(t)
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	if err := order.Transition(StatusCancelReq, now); err != nil {
		t.Fatal(err)
	}
	if err := order.Transition(StatusCancelled, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := order.Transition(StatusCompleted, now.Add(2*time.Minute)); !errors.Is(err, ErrTerminalOrder) {
		t.Fatalf("err=%v", err)
	}
	order = validOrder(t)
	if err := order.Transition(StatusCompleted, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("err=%v", err)
	}
	if err := order.Transition(StatusConfirmed, time.Time{}); !errors.Is(err, ErrInvalidTimestamp) {
		t.Fatalf("err=%v", err)
	}
}
