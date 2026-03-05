package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID
	Username       string
	HashedPassword string
	Version        int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}
