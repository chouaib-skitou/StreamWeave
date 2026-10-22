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
		{"order id", "bad", "usr_1", "EUR", []Item{{ProductID: "prd_1", Quantity: 1}}, ErrInvalidOrderID},
		{"customer id", "ord_1", "customer", "EUR", []Item{{ProductID: "prd_1", Quantity: 1}}, ErrInvalidCustomerID},
		{"currency", "ord_1", "usr_1", "EU", []Item{{ProductID: "prd_1", Quantity: 1}}, ErrInvalidCurrency},
		{"empty", "ord_1", "usr_1", "EUR", nil, ErrEmptyOrder},
		{"product id", "ord_1", "usr_1", "EUR", []Item{{ProductID: "product", Quantity: 1}}, ErrInvalidProductID},
		{"quantity", "ord_1", "usr_1", "EUR", []Item{{ProductID: "prd_1", Quantity: 0}}, ErrInvalidQuantity},
		{"duplicate", "ord_1", "usr_1", "EUR", []Item{{ProductID: "prd_1", Quantity: 1}, {ProductID: "prd_1", Quantity: 2}}, ErrDuplicateProduct},
		{"time", "ord_1", "usr_1", "EUR", []Item{{ProductID: "prd_1", Quantity: 1}}, ErrInvalidTimestamp},
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
	if err := order.ApplyQuote("qte_01", 1, map[string]int64{"prd_01": 1250}, now); err != nil {
		t.Fatal(err)
	}
	if err := order.Confirm(true, true, order.Version, now.Add(time.Minute)); err != nil {
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

func TestQuoteSnapshotsPricesAndRejectsOverflowOrMutation(t *testing.T) {
	order := validOrder(t)
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	if err := order.ApplyQuote("qte_1", 1, map[string]int64{"prd_01": 1250}, now); err != nil {
		t.Fatal(err)
	}
	if order.TotalMinor == nil || *order.TotalMinor != 2500 || order.Items[0].LineTotalMinor == nil || *order.Items[0].LineTotalMinor != 2500 {
		t.Fatalf("quote was not snapshotted: %+v", order)
	}
	if err := order.ApplyQuote("qte_2", 2, map[string]int64{"prd_01": 1}, now); !errors.Is(err, ErrPricesAlreadySet) {
		t.Fatalf("err=%v", err)
	}
	overflow := validOrder(t)
	if err := overflow.ApplyQuote("qte_1", 1, map[string]int64{"prd_01": MaxTotalMinor}, now); !errors.Is(err, ErrMoneyOverflow) {
		t.Fatalf("err=%v", err)
	}
}

func TestConfirmRequiresQuotePrerequisitesAndVersion(t *testing.T) {
	order := validOrder(t)
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	if err := order.Confirm(true, true, order.Version, now); !errors.Is(err, ErrMissingPrereq) {
		t.Fatalf("err=%v", err)
	}
	if err := order.ApplyQuote("qte_1", 1, map[string]int64{"prd_01": 10}, now); err != nil {
		t.Fatal(err)
	}
	if err := order.Confirm(true, true, order.Version-1, now); !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("err=%v", err)
	}
	if err := order.Confirm(true, false, order.Version, now); !errors.Is(err, ErrMissingPrereq) {
		t.Fatalf("err=%v", err)
	}
	if err := order.Confirm(true, true, order.Version, now); err != nil || order.Status != StatusConfirmed {
		t.Fatalf("status=%s err=%v", order.Status, err)
	}
}
