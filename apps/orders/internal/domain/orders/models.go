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
	MaxTotalMinor          = int64(1_000_000_000)
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
	ProductID      string
	Quantity       int32
	UnitPriceMinor *int64
	LineTotalMinor *int64
	QuoteID        *string
	QuoteVersion   *int64
}

type Order struct {
	ID                string
	CustomerID        string
	Currency          string
	Items             []Item
	TotalMinor        *int64
	ReservationID     *string
	PaymentID         *string
	ShipmentID        *string
	FailureCode       *string
	InventoryReleased bool
	PaymentRefunded   bool
	Status            Status
	Version           uint64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

var (
	orderIDPattern       = regexp.MustCompile(`^ord_[A-Za-z0-9]+$`)
	customerIDPattern    = regexp.MustCompile(`^usr_[A-Za-z0-9]+$`)
	productIDPattern     = regexp.MustCompile(`^prd_[A-Za-z0-9]+$`)
	quoteIDPattern       = regexp.MustCompile(`^qte_[A-Za-z0-9]+$`)
	reservationIDPattern = regexp.MustCompile(`^res_[A-Za-z0-9]+$`)
	paymentIDPattern     = regexp.MustCompile(`^pay_[A-Za-z0-9]+$`)
	shipmentIDPattern    = regexp.MustCompile(`^shp_[A-Za-z0-9]+$`)
	currencyPattern      = regexp.MustCompile(`^[A-Z]{3}$`)
)

func ValidOrderID(value string) bool       { return orderIDPattern.MatchString(value) }
func ValidCustomerID(value string) bool    { return customerIDPattern.MatchString(value) }
func ValidProductID(value string) bool     { return productIDPattern.MatchString(value) }
func ValidReservationID(value string) bool { return reservationIDPattern.MatchString(value) }
func ValidPaymentID(value string) bool     { return paymentIDPattern.MatchString(value) }
func ValidShipmentID(value string) bool    { return shipmentIDPattern.MatchString(value) }

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
	copyItems := make([]Item, len(items))
	for index, item := range items {
		copyItems[index] = Item{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	return Order{ID: id, CustomerID: customerID, Currency: currency, Items: copyItems, Status: StatusPending, Version: InitialVersion, CreatedAt: now, UpdatedAt: now}, nil
}

func (o *Order) Transition(to Status, now time.Time) error {
	return o.TransitionIfVersion(o.Version, to, now)
}

func (o *Order) TransitionIfVersion(expectedVersion uint64, to Status, now time.Time) error {
	if o == nil {
		return ErrInvalidTransition
	}
	if o.Version != expectedVersion {
		return ErrStaleVersion
	}
	if o.Status.IsTerminal() {
		return ErrTerminalOrder
	}
	if now.IsZero() || now.Location() != time.UTC {
		return ErrInvalidTimestamp
	}
	if to == StatusConfirmed {
		return ErrMissingPrereq
	}
	if !allowedTransition(o.Status, to) {
		return ErrInvalidTransition
	}
	o.Status = to
	o.Version++
	o.UpdatedAt = now
	return nil
}

func (o *Order) ApplyQuote(quoteID string, quoteVersion int64, prices map[string]int64, now time.Time) error {
	if o == nil {
		return ErrInvalidTransition
	}
	if o.Status != StatusPending || o.TotalMinor != nil {
		return ErrPricesAlreadySet
	}
	if !quoteIDPattern.MatchString(quoteID) || quoteVersion < 1 {
		return ErrInvalidMoney
	}
	if now.IsZero() || now.Location() != time.UTC {
		return ErrInvalidTimestamp
	}
	var total int64
	type quoteLine struct {
		price, lineTotal int64
	}
	lines := make([]quoteLine, len(o.Items))
	for index := range o.Items {
		item := o.Items[index]
		price, ok := prices[item.ProductID]
		if !ok || price < 0 || price > MaxTotalMinor {
			return ErrInvalidMoney
		}
		if price != 0 && int64(item.Quantity) > MaxTotalMinor/price {
			return ErrMoneyOverflow
		}
		lineTotal := int64(item.Quantity) * price
		if total > MaxTotalMinor-lineTotal {
			return ErrMoneyOverflow
		}
		total += lineTotal
		lines[index] = quoteLine{price: price, lineTotal: lineTotal}
	}
	for index := range o.Items {
		priceCopy, lineCopy := lines[index].price, lines[index].lineTotal
		quoteCopy, versionCopy := quoteID, quoteVersion
		o.Items[index].UnitPriceMinor, o.Items[index].LineTotalMinor = &priceCopy, &lineCopy
		o.Items[index].QuoteID, o.Items[index].QuoteVersion = &quoteCopy, &versionCopy
	}
	o.TotalMinor = &total
	o.Version++
	o.UpdatedAt = now
	return nil
}

func (o *Order) Confirm(inventoryReserved, paymentAuthorized bool, expectedVersion uint64, now time.Time) error {
	if !inventoryReserved || !paymentAuthorized {
		return ErrMissingPrereq
	}
	if o == nil || o.TotalMinor == nil {
		return ErrMissingPrereq
	}
	if o.Version != expectedVersion {
		return ErrStaleVersion
	}
	return o.transitionWithoutPrerequisite(StatusConfirmed, now)
}

func (o *Order) transitionWithoutPrerequisite(to Status, now time.Time) error {
	if o.Status.IsTerminal() {
		return ErrTerminalOrder
	}
	if now.IsZero() || now.Location() != time.UTC {
		return ErrInvalidTimestamp
	}
	if !allowedTransition(o.Status, to) {
		return ErrInvalidTransition
	}
	o.Status, o.Version, o.UpdatedAt = to, o.Version+1, now
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
