//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	adaphttp "github.com/ingunawandra/mini-wallet/internal/adapter/http"
	"github.com/ingunawandra/mini-wallet/internal/adapter/http/handler"
	"github.com/ingunawandra/mini-wallet/internal/adapter/repository"
	"github.com/ingunawandra/mini-wallet/internal/core/service"
	"github.com/ingunawandra/mini-wallet/internal/infrastructure/token"

	"github.com/gin-gonic/gin"
)

var (
	testPool   *pgxpool.Pool
	testRouter *gin.Engine
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("mini_wallet_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	// Run migrations — path relative to repo root where tests are invoked
	mig, err := migrate.New("file://../../../migrations", connStr)
	if err != nil {
		log.Fatalf("failed to create migrator: %v", err)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migrations: %v", err)
	}
	mig.Close()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("failed to create pool: %v", err)
	}
	testPool = pool
	testRouter = buildTestRouter(pool)

	code := m.Run()

	pool.Close()
	if err := pgContainer.Terminate(ctx); err != nil {
		log.Printf("failed to terminate container: %v", err)
	}

	os.Exit(code)
}

func buildTestRouter(pool *pgxpool.Pool) *gin.Engine {
	const (
		jwtSecret     = "test-secret-key"
		jwtExpiryMins = 60
		bcryptCost    = 4
	)

	tokenSvc := token.NewJWTService(jwtSecret, jwtExpiryMins)

	userRepo := repository.NewUserRepo(pool)
	walletRepo := repository.NewWalletRepo(pool)
	txRepo := repository.NewTxRepo(pool)
	txManager := repository.NewTxManager(pool)

	authSvc := service.NewAuthService(userRepo, walletRepo, txManager, tokenSvc, bcryptCost)
	walletSvc := service.NewWalletService(userRepo, walletRepo, txRepo, txManager)

	authHandler := handler.NewAuthHandler(authSvc)
	walletHandler := handler.NewWalletHandler(walletSvc)

	return adaphttp.NewRouter(authHandler, walletHandler, tokenSvc)
}

// truncate registers a cleanup that truncates all tables after each test.
func truncate(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		_, err := testPool.Exec(context.Background(),
			"TRUNCATE users, wallets, transactions CASCADE")
		require.NoError(t, err)
	})
}

// do sends an HTTP request to the test router and returns the recorder.
func do(method, path, body, authHeader string) *httptest.ResponseRecorder {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// registerUser registers a user and returns the raw response.
func registerUser(username, password string) *httptest.ResponseRecorder {
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	return do(http.MethodPost, "/api/v1/auth/register", body, "")
}

// loginAs registers (if needed) and logs in, returning the Bearer token string.
func loginAs(t *testing.T, username, password string) string {
	t.Helper()

	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	w := do(http.MethodPost, "/api/v1/auth/login", body, "")
	require.Equal(t, http.StatusOK, w.Code, "loginAs: login failed: %s", w.Body.String())

	// Extract access_token from response
	raw := w.Body.String()
	// simple extraction: find "access_token":"..."
	start := strings.Index(raw, `"access_token":"`)
	require.Greater(t, start, -1, "access_token not found in login response")
	start += len(`"access_token":"`)
	end := strings.Index(raw[start:], `"`)
	require.Greater(t, end, -1)
	tok := raw[start : start+end]
	return "Bearer " + tok
}
