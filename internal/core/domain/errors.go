package domain

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is a domain error that carries HTTP status, a machine-readable
// code, and a human-readable message. It implements the standard error
// interface so it can be used anywhere a plain error is expected.
//
// Sentinel variables below are *AppError pointers. Callers can use either:
//   - errors.Is(err, domain.ErrUserNotFound)   — identity check
//   - errors.As(err, &appErr)                  — to extract status/code
type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAppError constructs a new AppError. Prefer using the sentinel vars below.
func NewAppError(httpStatus int, code, message string) *AppError {
	return &AppError{HTTPStatus: httpStatus, Code: code, Message: message}
}

// Sentinel domain errors. Each is a *AppError so that:
//   - errors.Is(err, domain.ErrXxx) works via pointer equality
//   - errors.As(err, &appErr)       works to extract HTTP metadata
var (
	// Auth
	ErrUserNotFound = &AppError{
		HTTPStatus: http.StatusNotFound,
		Code:       "USER_NOT_FOUND",
		Message:    "user not found",
	}
	ErrUserAlreadyExists = &AppError{
		HTTPStatus: http.StatusConflict,
		Code:       "USERNAME_TAKEN",
		Message:    "username is already taken",
	}
	ErrInvalidCredentials = &AppError{
		HTTPStatus: http.StatusUnauthorized,
		Code:       "INVALID_CREDENTIALS",
		Message:    "invalid username or password",
	}
	// ErrInvalidToken is returned when a JWT cannot be parsed or has expired.
	// Distinct from ErrInvalidCredentials (login) — this is for Bearer token validation.
	ErrInvalidToken = &AppError{
		HTTPStatus: http.StatusUnauthorized,
		Code:       "INVALID_TOKEN",
		Message:    "invalid or expired token",
	}

	// Wallet
	ErrWalletNotFound = &AppError{
		HTTPStatus: http.StatusNotFound,
		Code:       "WALLET_NOT_FOUND",
		Message:    "wallet not found",
	}
	ErrInsufficientBalance = &AppError{
		HTTPStatus: http.StatusUnprocessableEntity,
		Code:       "INSUFFICIENT_BALANCE",
		Message:    "insufficient available balance",
	}
	ErrRecipientNotFound = &AppError{
		HTTPStatus: http.StatusNotFound,
		Code:       "RECIPIENT_NOT_FOUND",
		Message:    "recipient user not found",
	}

	// Transaction
	ErrDuplicateReference = &AppError{
		HTTPStatus: http.StatusConflict,
		Code:       "DUPLICATE_REFERENCE",
		Message:    "reference_id has already been processed",
	}

	// Concurrency
	ErrOptimisticLock = &AppError{
		HTTPStatus: http.StatusConflict,
		Code:       "LOCK_CONTENTION",
		Message:    "concurrent modification detected, please retry",
	}
)

// ErrInternalServer is a convenience constructor for unexpected failures.
// Unlike the sentinels above, each call produces a distinct *AppError so the
// original underlying error message can be included.
func ErrInternalServer(cause error) *AppError {
	msg := "an unexpected error occurred"
	if cause != nil {
		msg = cause.Error()
	}
	return &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       "INTERNAL_ERROR",
		Message:    msg,
	}
}

// IsAppError reports whether err is (or wraps) an *AppError.
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}
