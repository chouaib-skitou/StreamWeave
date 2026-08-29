package identity

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrRateLimited        = errors.New("rate limit exceeded")
	ErrExpired            = errors.New("expired")
	ErrAlreadyUsed        = errors.New("already used")
	ErrDependency         = errors.New("dependency unavailable")
)
