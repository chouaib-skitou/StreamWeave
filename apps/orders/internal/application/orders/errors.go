package orders

import "errors"

var (
	ErrInvalidDependency   = errors.New("orders application dependencies are required")
	ErrUnauthorized        = errors.New("orders authentication is required")
	ErrForbidden           = errors.New("orders operation is forbidden")
	ErrNotFound            = errors.New("order was not found")
	ErrInvalidInput        = errors.New("order input is invalid")
	ErrInvalidCursor       = errors.New("order cursor is invalid")
	ErrIdempotencyConflict = errors.New("idempotency key was reused with different input")
	ErrConflict            = errors.New("order operation conflicts with current state")
	ErrDependency          = errors.New("orders dependency is unavailable")
)
