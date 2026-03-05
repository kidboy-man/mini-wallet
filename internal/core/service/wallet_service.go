package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
	"github.com/shopspring/decimal"
)

const maxRetries = 3

type walletService struct {
	userRepo   port.UserRepository
	walletRepo port.WalletRepository
	txRepo     port.TransactionRepository
	txManager  port.TxManager
}

// NewWalletService creates a WalletService.
func NewWalletService(
	userRepo port.UserRepository,
	walletRepo port.WalletRepository,
	txRepo port.TransactionRepository,
	txManager port.TxManager,
) port.WalletService {
	return &walletService{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		txManager:  txManager,
	}
}

func (s *walletService) GetBalance(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	return s.walletRepo.FindByUserID(ctx, userID)
}

// TopUp credits the wallet atomically. Idempotent on referenceID.
func (s *walletService) TopUp(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, referenceID string) (*domain.Transaction, *domain.Wallet, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// Idempotency check for topup: use (wallet.ID as from_id surrogate, referenceID).
	// TopUps use to_id only, so we store wallet.ID as a logical reference using to_id approach.
	// We perform check before insert; unique constraint on DB is the final safety net.
	existing, err := s.txRepo.FindByFromIDAndReference(ctx, wallet.ID, referenceID)
	if err == domain.ErrDuplicateReference && existing != nil {
		updatedWallet, wErr := s.walletRepo.FindByUserID(ctx, userID)
		if wErr != nil {
			return existing, nil, wErr
		}
		return existing, updatedWallet, domain.ErrDuplicateReference
	}

	now := time.Now().UTC()
	tx := &domain.Transaction{
		ID:          uuid.New(),
		ToID:        &wallet.ID,
		ReferenceID: referenceID,
		Action:      domain.ActionTopup,
		Status:      domain.StatusPending,
		Amount:      amount,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	var updatedWallet *domain.Wallet

	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.txRepo.Create(txCtx, tx); err != nil {
			return err
		}

		wallet.Balance = wallet.Balance.Add(amount)
		if err := s.walletRepo.UpdateBalanceWithVersion(txCtx, wallet); err != nil {
			return err
		}
		wallet.Version++

		if err := s.txRepo.UpdateStatus(txCtx, tx.ID, domain.StatusSuccess, tx.Version); err != nil {
			return err
		}
		tx.Status = domain.StatusSuccess

		updatedWallet = wallet
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return tx, updatedWallet, nil
}

// Withdraw debits the wallet with optimistic locking and virtual balance lock.
// The operation is idempotent on referenceID.
func (s *walletService) Withdraw(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, referenceID string) (*domain.Transaction, *domain.Wallet, error) {
	var (
		resultTx     *domain.Transaction
		resultWallet *domain.Wallet
	)

	for attempt := 0; attempt < maxRetries; attempt++ {
		tx, wallet, err := s.doWithdraw(ctx, userID, amount, referenceID)
		if err == nil {
			resultTx = tx
			resultWallet = wallet
			break
		}
		if err == domain.ErrOptimisticLock {
			if attempt < maxRetries-1 {
				jitter := time.Duration(rand.Intn(50)) * time.Millisecond
				time.Sleep(time.Duration(attempt+1)*20*time.Millisecond + jitter)
				continue
			}
			return nil, nil, domain.ErrOptimisticLock
		}
		return nil, nil, err
	}

	return resultTx, resultWallet, nil
}

func (s *walletService) doWithdraw(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, referenceID string) (*domain.Transaction, *domain.Wallet, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	if wallet.AvailableBalance().LessThan(amount) {
		return nil, nil, domain.ErrInsufficientBalance
	}

	// Idempotency check
	existing, err := s.txRepo.FindByFromIDAndReference(ctx, wallet.ID, referenceID)
	if err == domain.ErrDuplicateReference && existing != nil {
		updatedWallet, wErr := s.walletRepo.FindByUserID(ctx, userID)
		if wErr != nil {
			return existing, nil, wErr
		}
		return existing, updatedWallet, domain.ErrDuplicateReference
	}

	now := time.Now().UTC()
	tx := &domain.Transaction{
		ID:          uuid.New(),
		FromID:      &wallet.ID,
		ReferenceID: referenceID,
		Action:      domain.ActionWithdraw,
		Status:      domain.StatusPending,
		Amount:      amount,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	var updatedWallet *domain.Wallet

	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.txRepo.Create(txCtx, tx); err != nil {
			return err
		}

		// Lock amount: increment locked_amount
		wallet.LockedAmount = wallet.LockedAmount.Add(amount)
		if err := s.walletRepo.UpdateBalanceWithVersion(txCtx, wallet); err != nil {
			return err
		}
		wallet.Version++

		// Settle: debit balance and release lock
		wallet.Balance = wallet.Balance.Sub(amount)
		wallet.LockedAmount = wallet.LockedAmount.Sub(amount)
		if err := s.walletRepo.UpdateBalanceWithVersion(txCtx, wallet); err != nil {
			return err
		}
		wallet.Version++

		if err := s.txRepo.UpdateStatus(txCtx, tx.ID, domain.StatusSuccess, tx.Version); err != nil {
			return err
		}
		tx.Status = domain.StatusSuccess

		updatedWallet = wallet
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return tx, updatedWallet, nil
}

// Transfer moves funds from one user to another atomically.
func (s *walletService) Transfer(ctx context.Context, fromUserID uuid.UUID, toUsername string, amount decimal.Decimal, referenceID string) (*domain.Transaction, *domain.Wallet, error) {
	recipient, err := s.userRepo.FindByUsername(ctx, toUsername)
	if err != nil {
		return nil, nil, domain.ErrRecipientNotFound
	}

	var (
		resultTx     *domain.Transaction
		resultWallet *domain.Wallet
	)

	for attempt := 0; attempt < maxRetries; attempt++ {
		tx, wallet, err := s.doTransfer(ctx, fromUserID, recipient.ID, amount, referenceID)
		if err == nil {
			resultTx = tx
			resultWallet = wallet
			break
		}
		if err == domain.ErrOptimisticLock {
			if attempt < maxRetries-1 {
				jitter := time.Duration(rand.Intn(50)) * time.Millisecond
				time.Sleep(time.Duration(attempt+1)*20*time.Millisecond + jitter)
				continue
			}
			return nil, nil, domain.ErrOptimisticLock
		}
		return nil, nil, err
	}

	return resultTx, resultWallet, nil
}

func (s *walletService) doTransfer(ctx context.Context, fromUserID, toUserID uuid.UUID, amount decimal.Decimal, referenceID string) (*domain.Transaction, *domain.Wallet, error) {
	fromWallet, err := s.walletRepo.FindByUserID(ctx, fromUserID)
	if err != nil {
		return nil, nil, err
	}

	if fromWallet.AvailableBalance().LessThan(amount) {
		return nil, nil, domain.ErrInsufficientBalance
	}

	toWallet, err := s.walletRepo.FindByUserID(ctx, toUserID)
	if err != nil {
		return nil, nil, err
	}

	// Idempotency check
	existing, err := s.txRepo.FindByFromIDAndReference(ctx, fromWallet.ID, referenceID)
	if err == domain.ErrDuplicateReference && existing != nil {
		updatedWallet, wErr := s.walletRepo.FindByUserID(ctx, fromUserID)
		if wErr != nil {
			return existing, nil, wErr
		}
		return existing, updatedWallet, domain.ErrDuplicateReference
	}

	now := time.Now().UTC()
	tx := &domain.Transaction{
		ID:          uuid.New(),
		FromID:      &fromWallet.ID,
		ToID:        &toWallet.ID,
		ReferenceID: referenceID,
		Action:      domain.ActionTransfer,
		Status:      domain.StatusPending,
		Amount:      amount,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	var updatedWallet *domain.Wallet

	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.txRepo.Create(txCtx, tx); err != nil {
			return err
		}

		// Lock sender amount
		fromWallet.LockedAmount = fromWallet.LockedAmount.Add(amount)
		if err := s.walletRepo.UpdateBalanceWithVersion(txCtx, fromWallet); err != nil {
			return err
		}
		fromWallet.Version++

		// Settle sender: debit balance and release lock
		fromWallet.Balance = fromWallet.Balance.Sub(amount)
		fromWallet.LockedAmount = fromWallet.LockedAmount.Sub(amount)
		if err := s.walletRepo.UpdateBalanceWithVersion(txCtx, fromWallet); err != nil {
			return err
		}
		fromWallet.Version++

		// Credit receiver
		toWallet.Balance = toWallet.Balance.Add(amount)
		if err := s.walletRepo.UpdateBalanceWithVersion(txCtx, toWallet); err != nil {
			return err
		}

		if err := s.txRepo.UpdateStatus(txCtx, tx.ID, domain.StatusSuccess, tx.Version); err != nil {
			return err
		}
		tx.Status = domain.StatusSuccess

		updatedWallet = fromWallet
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return tx, updatedWallet, nil
}
