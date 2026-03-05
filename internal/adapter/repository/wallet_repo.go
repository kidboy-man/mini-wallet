package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
	infradb "github.com/ingunawandra/mini-wallet/internal/infrastructure/db"
)

type walletRepo struct {
	pool *pgxpool.Pool
}

// NewWalletRepo creates a WalletRepository backed by PostgreSQL.
func NewWalletRepo(pool *pgxpool.Pool) port.WalletRepository {
	return &walletRepo{pool: pool}
}

func (r *walletRepo) Create(ctx context.Context, wallet *domain.Wallet) error {
	q := infradb.GetQuerier(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO wallets (id, user_id, balance, locked_amount, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		wallet.ID, wallet.UserID, wallet.Balance, wallet.LockedAmount, wallet.Version, wallet.CreatedAt, wallet.UpdatedAt,
	)
	return err
}

func (r *walletRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	q := infradb.GetQuerier(ctx, r.pool)
	row := q.QueryRow(ctx,
		`SELECT id, user_id, balance, locked_amount, version, created_at, updated_at
		 FROM wallets WHERE user_id = $1`,
		userID,
	)
	return scanWallet(row)
}

func (r *walletRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Wallet, error) {
	q := infradb.GetQuerier(ctx, r.pool)
	row := q.QueryRow(ctx,
		`SELECT id, user_id, balance, locked_amount, version, created_at, updated_at
		 FROM wallets WHERE id = $1`,
		id,
	)
	return scanWallet(row)
}

func (r *walletRepo) UpdateBalanceWithVersion(ctx context.Context, wallet *domain.Wallet) error {
	q := infradb.GetQuerier(ctx, r.pool)
	tag, err := q.Exec(ctx,
		`UPDATE wallets
		 SET balance = $1, locked_amount = $2, version = version + 1, updated_at = NOW()
		 WHERE id = $3 AND version = $4`,
		wallet.Balance, wallet.LockedAmount, wallet.ID, wallet.Version,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOptimisticLock
	}
	return nil
}

func scanWallet(row pgx.Row) (*domain.Wallet, error) {
	w := &domain.Wallet{}
	err := row.Scan(&w.ID, &w.UserID, &w.Balance, &w.LockedAmount, &w.Version, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		return nil, err
	}
	return w, nil
}
