package model

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("access denied")
	ErrForbidden    = errors.New("forbidden")
	ErrInternal     = errors.New("internal system error")
)
