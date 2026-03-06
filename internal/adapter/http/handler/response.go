package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
)

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

// domainErrToHTTP extracts HTTP status and error code directly from an
// *AppError. If the error is not an AppError, falls back to 500.
func domainErrToHTTP(c *gin.Context, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}
	fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}
