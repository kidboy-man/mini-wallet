package domain

import "errors"

var (
	// Auth errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")

	// Wallet errors
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientBalance = errors.New("insufficient available balance")
	ErrRecipientNotFound   = errors.New("recipient not found")

	// Transaction errors
	ErrDuplicateReference = errors.New("duplicate reference_id for this wallet")

	// Concurrency
	ErrOptimisticLock = errors.New("concurrent modification detected, please retry")
)
