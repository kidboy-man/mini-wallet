package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ingunawandra/mini-wallet/config"
)

type contextKey string

const txKey contextKey = "pgx_tx"

// NewPool creates and validates a pgxpool connection.
func NewPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}
	return pool, nil
}

// WithTx executes fn inside a database transaction.
// The transaction is stored in the context so repositories can retrieve it.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txKey, tx)

	if fnErr := fn(txCtx); fnErr != nil {
		_ = tx.Rollback(ctx)
		return fnErr
	}

	return tx.Commit(ctx)
}

// ExtractTx retrieves the pgx.Tx from context if present.
func ExtractTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// Querier is the common interface for pgx.Tx and pgxpool.Pool.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// GetQuerier returns the transaction from context if present, else falls back to the pool.
func GetQuerier(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx := ExtractTx(ctx); tx != nil {
		return tx
	}
	return pool
}
