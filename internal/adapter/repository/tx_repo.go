package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
	infradb "github.com/ingunawandra/mini-wallet/internal/infrastructure/db"
)

type txRepo struct {
	pool *pgxpool.Pool
}

// NewTxRepo creates a TransactionRepository backed by PostgreSQL.
func NewTxRepo(pool *pgxpool.Pool) port.TransactionRepository {
	return &txRepo{pool: pool}
}

func (r *txRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	q := infradb.GetQuerier(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO transactions
		 (id, from_id, to_id, reference_id, parent_transaction_id, action, status, amount, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		tx.ID, tx.FromID, tx.ToID, tx.ReferenceID, tx.ParentTransactionID,
		tx.Action, tx.Status, tx.Amount, tx.Version, tx.CreatedAt, tx.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateReference
		}
		return domain.ErrInternalServer(err)
	}
	return nil
}

func (r *txRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TransactionStatus, currentVersion int) error {
	q := infradb.GetQuerier(ctx, r.pool)
	tag, err := q.Exec(ctx,
		`UPDATE transactions SET status = $1, version = version + 1, updated_at = NOW()
		 WHERE id = $2 AND version = $3`,
		status, id, currentVersion,
	)
	if err != nil {
		return domain.ErrInternalServer(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOptimisticLock
	}
	return nil
}

func (r *txRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	q := infradb.GetQuerier(ctx, r.pool)
	row := q.QueryRow(ctx,
		`SELECT id, from_id, to_id, reference_id, parent_transaction_id, action, status, amount, version, created_at, updated_at
		 FROM transactions WHERE id = $1`,
		id,
	)
	return scanTransaction(row)
}

func (r *txRepo) FindByFromIDAndReference(ctx context.Context, fromID uuid.UUID, referenceID string) (*domain.Transaction, error) {
	q := infradb.GetQuerier(ctx, r.pool)
	row := q.QueryRow(ctx,
		`SELECT id, from_id, to_id, reference_id, parent_transaction_id, action, status, amount, version, created_at, updated_at
		 FROM transactions WHERE from_id = $1 AND reference_id = $2`,
		fromID, referenceID,
	)
	tx, err := scanTransaction(row)
	if err != nil {
		return nil, err
	}
	return tx, domain.ErrDuplicateReference
}

func scanTransaction(row pgx.Row) (*domain.Transaction, error) {
	t := &domain.Transaction{}
	err := row.Scan(
		&t.ID, &t.FromID, &t.ToID, &t.ReferenceID, &t.ParentTransactionID,
		&t.Action, &t.Status, &t.Amount, &t.Version, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, domain.ErrInternalServer(err)
	}
	return t, nil
}
