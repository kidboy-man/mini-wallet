package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// WalletRepository defines persistence operations for wallets.
type WalletRepository interface {
	Create(ctx context.Context, wallet *domain.Wallet) error
	FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Wallet, error)
	// UpdateBalanceWithVersion performs an optimistic-lock update.
	// Updates balance, locked_amount, version (version+1), updated_at WHERE id=$id AND version=$currentVersion.
	// Returns domain.ErrOptimisticLock if no rows were affected.
	UpdateBalanceWithVersion(ctx context.Context, wallet *domain.Wallet) error
}

// TransactionRepository defines persistence operations for transactions.
type TransactionRepository interface {
	Create(ctx context.Context, tx *domain.Transaction) error
	// UpdateStatus updates status and increments version WHERE id=$id AND version=$currentVersion.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TransactionStatus, currentVersion int) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	// FindByFromIDAndReference returns the existing transaction for idempotency check.
	// Returns domain.ErrDuplicateReference if a matching record exists.
	FindByFromIDAndReference(ctx context.Context, fromID uuid.UUID, referenceID string) (*domain.Transaction, error)
}

// TxManager abstracts database transaction management.
// The callback receives a context that carries the active DB transaction.
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
