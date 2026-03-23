package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors.
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrUnavailable  = errors.New("unavailable")
)

// AppError is an error with an HTTP-friendly status code.
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// New wraps err with a status code and message.
func New(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// Is and As delegate to the standard library.
func Is(err, target error) bool         { return errors.Is(err, target) }
func As(err error, target interface{}) bool { return errors.As(err, &target) }
