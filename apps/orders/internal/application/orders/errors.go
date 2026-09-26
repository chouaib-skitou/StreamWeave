package orders

import "errors"

var (
	ErrInvalidDependency = errors.New("orders application dependencies are required")
	ErrUnauthorized      = errors.New("orders authentication is required")
	ErrForbidden         = errors.New("orders operation is forbidden")
	ErrNotFound          = errors.New("order was not found")
	ErrInvalidInput      = errors.New("order input is invalid")
)
