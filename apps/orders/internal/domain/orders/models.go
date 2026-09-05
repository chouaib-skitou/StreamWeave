package orders

import (
	"regexp"
	"strings"
	"time"
)

const (
	MaxItems               = 100
	MaxQuantity            = 1000
	MaxCurrency            = 3
	InitialVersion         = uint64(1)
	StatusPending   Status = "PENDING"
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelReq Status = "CANCEL_REQUESTED"
	StatusCancelled Status = "CANCELLED"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
)

type Status string

func (s Status) IsTerminal() bool {
	return s == StatusCancelled || s == StatusCompleted || s == StatusFailed
}

type Item struct {
	ProductID string
	Quantity  int32
}

type Order struct {
	ID         string
	CustomerID string
	Currency   string
	Items      []Item
	TotalMinor *int64
	Status     Status
	Version    uint64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

var (
	orderIDPattern    = regexp.MustCompile(`^ord_[A-Za-z0-9]+$`)
	customerIDPattern = regexp.MustCompile(`^usr_[A-Za-z0-9]+$`)
	productIDPattern  = regexp.MustCompile(`^prd_[A-Za-z0-9]+$`)
	currencyPattern   = regexp.MustCompile(`^[A-Z]{3}$`)
)

func NewOrder(id, customerID, currency string, items []Item, now time.Time) (Order, error) {
	if !orderIDPattern.MatchString(id) {
		return Order{}, ErrInvalidOrderID
	}
	if !customerIDPattern.MatchString(customerID) {
		return Order{}, ErrInvalidCustomerID
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !currencyPattern.MatchString(currency) {
		return Order{}, ErrInvalidCurrency
	}
	if len(items) == 0 {
		return Order{}, ErrEmptyOrder
	}
	if len(items) > MaxItems {
		return Order{}, ErrTooManyItems
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if !productIDPattern.MatchString(item.ProductID) {
			return Order{}, ErrInvalidProductID
		}
		if item.Quantity < 1 || item.Quantity > MaxQuantity {
			return Order{}, ErrInvalidQuantity
		}
		if _, exists := seen[item.ProductID]; exists {
			return Order{}, ErrDuplicateProduct
		}
		seen[item.ProductID] = struct{}{}
	}
	if now.IsZero() || now.Location() != time.UTC {
		return Order{}, ErrInvalidTimestamp
	}
	copyItems := append([]Item(nil), items...)
	return Order{ID: id, CustomerID: customerID, Currency: currency, Items: copyItems, Status: StatusPending, Version: InitialVersion, CreatedAt: now, UpdatedAt: now}, nil
}

func (o *Order) Transition(to Status, now time.Time) error {
	if o == nil {
		return ErrInvalidTransition
	}
	if o.Status.IsTerminal() {
		return ErrTerminalOrder
	}
	if now.IsZero() || now.Location() != time.UTC {
		return ErrInvalidTimestamp
	}
	if !allowedTransition(o.Status, to) {
		return ErrInvalidTransition
	}
	o.Status = to
	o.Version++
	o.UpdatedAt = now
	return nil
}

func allowedTransition(from, to Status) bool {
	switch from {
	case StatusPending:
		return to == StatusConfirmed || to == StatusCancelReq || to == StatusFailed
	case StatusConfirmed:
		return to == StatusCancelReq || to == StatusCompleted || to == StatusFailed
	case StatusCancelReq:
		return to == StatusCancelled || to == StatusFailed
	default:
		return false
	}
}
