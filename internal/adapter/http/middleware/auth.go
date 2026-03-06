package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
)

const (
	UserIDKey   = "user_id"
	UsernameKey = "username"
)

// Auth returns a Gin middleware that validates JWT Bearer tokens.
// All error responses use the same AppError envelope as the rest of the API.
func Auth(tokenService port.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortWithAppError(c, domain.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			abortWithAppError(c, domain.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format"))
			return
		}

		claims, err := tokenService.Validate(parts[1])
		if err != nil {
			var appErr *domain.AppError
			if errors.As(err, &appErr) {
				abortWithAppError(c, appErr)
			} else {
				abortWithAppError(c, domain.ErrInternalServer(err))
			}
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Next()
	}
}

func abortWithAppError(c *gin.Context, appErr *domain.AppError) {
	c.AbortWithStatusJSON(appErr.HTTPStatus, gin.H{
		"success": false,
		"error": gin.H{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	})
}
