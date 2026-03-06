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

type userRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a UserRepository backed by PostgreSQL.
func NewUserRepo(pool *pgxpool.Pool) port.UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	q := infradb.GetQuerier(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO users (id, username, hashed_password, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Username, user.HashedPassword, user.Version, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return domain.ErrInternalServer(err)
	}
	return nil
}

func (r *userRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	q := infradb.GetQuerier(ctx, r.pool)
	row := q.QueryRow(ctx,
		`SELECT id, username, hashed_password, version, created_at, updated_at, deleted_at
		 FROM users WHERE LOWER(username) = LOWER($1) AND deleted_at IS NULL`,
		username,
	)

	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Username, &u.HashedPassword, &u.Version, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, domain.ErrInternalServer(err)
	}
	return u, nil
}

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	q := infradb.GetQuerier(ctx, r.pool)
	row := q.QueryRow(ctx,
		`SELECT id, username, hashed_password, version, created_at, updated_at, deleted_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)

	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Username, &u.HashedPassword, &u.Version, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, domain.ErrInternalServer(err)
	}
	return u, nil
}

func (r *userRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	q := infradb.GetQuerier(ctx, r.pool)
	tag, err := q.Exec(ctx,
		`UPDATE users SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return domain.ErrInternalServer(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
