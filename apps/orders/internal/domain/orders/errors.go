package orders

import "errors"

var (
	ErrInvalidOrderID    = errors.New("invalid order id")
	ErrInvalidCustomerID = errors.New("invalid customer id")
	ErrInvalidProductID  = errors.New("invalid product id")
	ErrInvalidCurrency   = errors.New("invalid currency")
	ErrEmptyOrder        = errors.New("order must contain at least one item")
	ErrTooManyItems      = errors.New("order contains too many items")
	ErrInvalidQuantity   = errors.New("item quantity is invalid")
	ErrDuplicateProduct  = errors.New("order contains a duplicate product")
	ErrInvalidTransition = errors.New("invalid order transition")
	ErrTerminalOrder     = errors.New("terminal order cannot transition")
	ErrInvalidTimestamp  = errors.New("order timestamp must be UTC")
	ErrInvalidMoney      = errors.New("money amount is invalid")
	ErrMoneyOverflow     = errors.New("money amount exceeds order limit")
	ErrPricesAlreadySet  = errors.New("order prices are already set")
	ErrMissingPrereq     = errors.New("order confirmation prerequisites are missing")
	ErrStaleVersion      = errors.New("order aggregate version is stale")
)
