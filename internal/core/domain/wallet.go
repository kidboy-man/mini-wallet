package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Wallet struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Balance      decimal.Decimal
	LockedAmount decimal.Decimal
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// AvailableBalance returns the spendable balance (balance - locked_amount).
func (w *Wallet) AvailableBalance() decimal.Decimal {
	return w.Balance.Sub(w.LockedAmount)
}
