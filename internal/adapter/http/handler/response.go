package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
)

// Response is the standard JSON envelope.
type Response[T any] struct {
	Success bool      `json:"success"`
	Data    T         `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// APIError is the machine-readable error payload.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func success(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
	})
}

func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// domainErrToHTTP maps domain errors to HTTP status + error codes.
func domainErrToHTTP(c *gin.Context, err error) {
	switch err {
	case domain.ErrUserAlreadyExists:
		fail(c, http.StatusConflict, "USERNAME_TAKEN", "username is already taken")
	case domain.ErrInvalidCredentials:
		fail(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
	case domain.ErrInsufficientBalance:
		fail(c, http.StatusUnprocessableEntity, "INSUFFICIENT_BALANCE", "insufficient available balance")
	case domain.ErrDuplicateReference:
		fail(c, http.StatusConflict, "DUPLICATE_REFERENCE", "reference_id has already been processed")
	case domain.ErrOptimisticLock:
		fail(c, http.StatusConflict, "LOCK_CONTENTION", "concurrent modification detected, please retry")
	case domain.ErrRecipientNotFound:
		fail(c, http.StatusNotFound, "RECIPIENT_NOT_FOUND", "recipient user not found")
	case domain.ErrWalletNotFound:
		fail(c, http.StatusNotFound, "WALLET_NOT_FOUND", "wallet not found")
	default:
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}
