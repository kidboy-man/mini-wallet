package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
	infradb "github.com/ingunawandra/mini-wallet/internal/infrastructure/db"
)

type txManager struct {
	pool *pgxpool.Pool
}

// NewTxManager creates a TxManager backed by PostgreSQL.
func NewTxManager(pool *pgxpool.Pool) port.TxManager {
	return &txManager{pool: pool}
}

// WithTx runs fn inside a database transaction. Errors from the fn callback
// are passed through as-is (they are already *AppError from the domain/service
// layer). Errors from the infrastructure itself (begin/commit failures) are
// wrapped as ErrInternalServer so every caller always receives an *AppError.
func (m *txManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	err := infradb.WithTx(ctx, m.pool, fn)
	if err == nil {
		return nil
	}

	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		return err
	}

	return domain.ErrInternalServer(err)
}
