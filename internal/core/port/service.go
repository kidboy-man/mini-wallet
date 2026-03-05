package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/shopspring/decimal"
)

// AuthService defines application-level auth operations.
type AuthService interface {
	Register(ctx context.Context, username, password string) (*domain.User, error)
	Login(ctx context.Context, username, password string) (token string, expiresAt int64, err error)
}

// WalletService defines application-level wallet operations.
type WalletService interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error)
	TopUp(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, referenceID string) (*domain.Transaction, *domain.Wallet, error)
	Withdraw(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, referenceID string) (*domain.Transaction, *domain.Wallet, error)
	Transfer(ctx context.Context, fromUserID uuid.UUID, toUsername string, amount decimal.Decimal, referenceID string) (*domain.Transaction, *domain.Wallet, error)
}

// TokenService defines JWT token operations.
type TokenService interface {
	Generate(userID uuid.UUID, username string) (token string, expiresAt int64, err error)
	Validate(token string) (*TokenClaims, error)
}

// TokenClaims holds the parsed JWT claims.
type TokenClaims struct {
	UserID   uuid.UUID
	Username string
}
