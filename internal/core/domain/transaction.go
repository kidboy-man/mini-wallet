package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionAction string
type TransactionStatus string

const (
	ActionTopup    TransactionAction = "TOPUP"
	ActionWithdraw TransactionAction = "WITHDRAW"
	ActionTransfer TransactionAction = "TRANSFER"
	ActionRefund   TransactionAction = "REFUND"
)

const (
	StatusPending  TransactionStatus = "PENDING"
	StatusSuccess  TransactionStatus = "SUCCESS"
	StatusFailed   TransactionStatus = "FAILED"
	StatusReversed TransactionStatus = "REVERSED"
)

type Transaction struct {
	ID                  uuid.UUID
	FromID              *uuid.UUID
	ToID                *uuid.UUID
	ReferenceID         string
	ParentTransactionID *uuid.UUID
	Action              TransactionAction
	Status              TransactionStatus
	Amount              decimal.Decimal
	Version             int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
