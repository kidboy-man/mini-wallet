package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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

func (m *txManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return infradb.WithTx(ctx, m.pool, fn)
}
