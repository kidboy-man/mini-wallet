package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/ingunawandra/mini-wallet/config"
	_ "github.com/ingunawandra/mini-wallet/docs"
	adaphttp "github.com/ingunawandra/mini-wallet/internal/adapter/http"
	"github.com/ingunawandra/mini-wallet/internal/adapter/http/handler"
	"github.com/ingunawandra/mini-wallet/internal/adapter/repository"
	"github.com/ingunawandra/mini-wallet/internal/core/service"
	infradb "github.com/ingunawandra/mini-wallet/internal/infrastructure/db"
	"github.com/ingunawandra/mini-wallet/internal/infrastructure/token"
)

// @title Mini Wallet API
// @version 1.0
// @description REST API for user authentication and wallet operations.
// @BasePath /api/v1
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	ctx := context.Background()

	// Connect to database
	pool, err := infradb.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Run migrations
	if err := runMigrations(cfg.DBURL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Infrastructure
	tokenSvc := token.NewJWTService(cfg.JWTSecret, cfg.JWTExpiryMins)

	// Repositories
	userRepo := repository.NewUserRepo(pool)
	walletRepo := repository.NewWalletRepo(pool)
	txRepo := repository.NewTxRepo(pool)
	txManager := repository.NewTxManager(pool)

	// Services
	authSvc := service.NewAuthService(userRepo, walletRepo, txManager, tokenSvc, cfg.BcryptCost)
	walletSvc := service.NewWalletService(userRepo, walletRepo, txRepo, txManager)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	walletHandler := handler.NewWalletHandler(walletSvc)

	// Router
	router := adaphttp.NewRouter(authHandler, walletHandler, tokenSvc)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func runMigrations(dbURL string) error {
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Println("Migrations applied successfully")
	return nil
}
