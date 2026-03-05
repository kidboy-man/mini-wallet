package http

import (
	"github.com/gin-gonic/gin"
	"github.com/ingunawandra/mini-wallet/internal/adapter/http/handler"
	"github.com/ingunawandra/mini-wallet/internal/adapter/http/middleware"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
)

// NewRouter builds and returns the Gin router with all routes registered.
func NewRouter(
	authHandler *handler.AuthHandler,
	walletHandler *handler.WalletHandler,
	tokenService port.TokenService,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	v1 := r.Group("/api/v1")

	// Public auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Protected wallet routes
	wallets := v1.Group("/wallets")
	wallets.Use(middleware.Auth(tokenService))
	{
		wallets.GET("/me/balance", walletHandler.GetBalance)
		wallets.POST("/topup", walletHandler.TopUp)
		wallets.POST("/withdraw", walletHandler.Withdraw)
		wallets.POST("/transfer", walletHandler.Transfer)
	}

	return r
}
