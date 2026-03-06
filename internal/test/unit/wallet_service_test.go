package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
	"github.com/ingunawandra/mini-wallet/internal/core/port/mocks"
	"github.com/ingunawandra/mini-wallet/internal/core/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newWalletService(ctrl *gomock.Controller) (
	*mocks.MockUserRepository,
	*mocks.MockWalletRepository,
	*mocks.MockTransactionRepository,
	*mocks.MockTxManager,
	port.WalletService,
) {
	userRepo := mocks.NewMockUserRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)
	txRepo := mocks.NewMockTransactionRepository(ctrl)
	txManager := mocks.NewMockTxManager(ctrl)
	svc := service.NewWalletService(userRepo, walletRepo, txRepo, txManager)
	return userRepo, walletRepo, txRepo, txManager, svc
}

func withTxPassthrough(txManager *mocks.MockTxManager) {
	txManager.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
}

func TestWalletService_GetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	_, walletRepo, _, _, svc := newWalletService(ctrl)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &domain.Wallet{
		ID:           walletID,
		UserID:       userID,
		Balance:      decimal.NewFromFloat(1000),
		LockedAmount: decimal.NewFromFloat(50),
		Version:      1,
	}

	walletRepo.EXPECT().FindByUserID(ctx, userID).Return(wallet, nil)

	result, err := svc.GetBalance(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, wallet.Balance, result.Balance)
	assert.Equal(t, decimal.NewFromFloat(950), result.AvailableBalance())
}

func TestWalletService_TopUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	_, walletRepo, txRepo, txManager, svc := newWalletService(ctrl)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	amount := decimal.NewFromFloat(500)

	t.Run("success", func(t *testing.T) {
		wallet := &domain.Wallet{
			ID:           walletID,
			UserID:       userID,
			Balance:      decimal.NewFromFloat(1000),
			LockedAmount: decimal.Zero,
			Version:      1,
		}

		walletRepo.EXPECT().FindByUserID(ctx, userID).Return(wallet, nil)
		txRepo.EXPECT().FindByToIDAndReference(ctx, walletID, "ref-001").Return(nil, nil)
		withTxPassthrough(txManager)
		txRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		walletRepo.EXPECT().UpdateBalanceWithVersion(gomock.Any(), gomock.Any()).Return(nil)
		txRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), domain.StatusSuccess, 1).Return(nil)

		tx, updatedWallet, err := svc.TopUp(ctx, userID, amount, "ref-001")
		assert.NoError(t, err)
		assert.Equal(t, domain.ActionTopup, tx.Action)
		assert.Equal(t, decimal.NewFromFloat(1500), updatedWallet.Balance)
	})

	t.Run("idempotent - duplicate reference", func(t *testing.T) {
		wallet := &domain.Wallet{
			ID:           walletID,
			UserID:       userID,
			Balance:      decimal.NewFromFloat(1500),
			LockedAmount: decimal.Zero,
			Version:      2,
		}
		existingTx := &domain.Transaction{
			ID:     uuid.New(),
			Action: domain.ActionTopup,
			Status: domain.StatusSuccess,
		}

		walletRepo.EXPECT().FindByUserID(ctx, userID).Return(wallet, nil)
		txRepo.EXPECT().FindByToIDAndReference(ctx, walletID, "ref-001").Return(existingTx, domain.ErrDuplicateReference)
		walletRepo.EXPECT().FindByUserID(ctx, userID).Return(wallet, nil)

		tx, updatedWallet, err := svc.TopUp(ctx, userID, amount, "ref-001")
		assert.ErrorIs(t, err, domain.ErrDuplicateReference)
		assert.Equal(t, existingTx.ID, tx.ID)
		assert.Equal(t, decimal.NewFromFloat(1500), updatedWallet.Balance)
	})
}

func TestWalletService_Withdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	_, walletRepo, txRepo, txManager, svc := newWalletService(ctrl)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()

	t.Run("success", func(t *testing.T) {
		wallet := &domain.Wallet{
			ID:           walletID,
			UserID:       userID,
			Balance:      decimal.NewFromFloat(1000),
			LockedAmount: decimal.Zero,
			Version:      1,
		}
		amount := decimal.NewFromFloat(200)

		walletRepo.EXPECT().FindByUserID(ctx, userID).Return(wallet, nil)
		txRepo.EXPECT().FindByFromIDAndReference(ctx, walletID, "wd-001").Return(nil, nil)
		withTxPassthrough(txManager)
		txRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		// lock: locked_amount += 200
		walletRepo.EXPECT().UpdateBalanceWithVersion(gomock.Any(), gomock.Any()).Return(nil)
		// settle: balance -= 200, locked_amount -= 200
		walletRepo.EXPECT().UpdateBalanceWithVersion(gomock.Any(), gomock.Any()).Return(nil)
		txRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), domain.StatusSuccess, 1).Return(nil)

		tx, updatedWallet, err := svc.Withdraw(ctx, userID, amount, "wd-001")
		assert.NoError(t, err)
		assert.Equal(t, domain.ActionWithdraw, tx.Action)
		assert.True(t, decimal.NewFromFloat(800).Equal(updatedWallet.Balance))
		assert.True(t, updatedWallet.LockedAmount.IsZero())
	})

	t.Run("insufficient balance", func(t *testing.T) {
		wallet := &domain.Wallet{
			ID:           walletID,
			UserID:       userID,
			Balance:      decimal.NewFromFloat(100),
			LockedAmount: decimal.Zero,
			Version:      1,
		}
		walletRepo.EXPECT().FindByUserID(ctx, userID).Return(wallet, nil)

		_, _, err := svc.Withdraw(ctx, userID, decimal.NewFromFloat(500), "wd-002")
		assert.ErrorIs(t, err, domain.ErrInsufficientBalance)
	})

	t.Run("optimistic lock exhausted after 3 retries", func(t *testing.T) {
		wallet := &domain.Wallet{
			ID:           walletID,
			UserID:       userID,
			Balance:      decimal.NewFromFloat(1000),
			LockedAmount: decimal.Zero,
			Version:      1,
		}
		amount := decimal.NewFromFloat(200)

		for i := 0; i < 3; i++ {
			walletRepo.EXPECT().FindByUserID(ctx, userID).Return(wallet, nil)
			txRepo.EXPECT().FindByFromIDAndReference(ctx, walletID, "wd-003").Return(nil, nil)
			withTxPassthrough(txManager)
			txRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			walletRepo.EXPECT().UpdateBalanceWithVersion(gomock.Any(), gomock.Any()).Return(domain.ErrOptimisticLock)
		}

		_, _, err := svc.Withdraw(ctx, userID, amount, "wd-003")
		assert.ErrorIs(t, err, domain.ErrOptimisticLock)
	})
}

func TestWalletService_Transfer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo, walletRepo, txRepo, txManager, svc := newWalletService(ctrl)
	ctx := context.Background()

	fromUserID := uuid.New()
	toUserID := uuid.New()
	fromWalletID := uuid.New()
	toWalletID := uuid.New()

	t.Run("success", func(t *testing.T) {
		fromWallet := &domain.Wallet{
			ID: fromWalletID, UserID: fromUserID,
			Balance: decimal.NewFromFloat(1000), LockedAmount: decimal.Zero, Version: 1,
		}
		toWallet := &domain.Wallet{
			ID: toWalletID, UserID: toUserID,
			Balance: decimal.Zero, LockedAmount: decimal.Zero, Version: 1,
		}
		recipient := &domain.User{ID: toUserID, Username: "bob"}
		amount := decimal.NewFromFloat(100)

		// doTransfer call order: FindByUsername → FindByUserID(from) → FindByUserID(to) → FindByFromIDAndReference → WithTx
		userRepo.EXPECT().FindByUsername(ctx, "bob").Return(recipient, nil)
		walletRepo.EXPECT().FindByUserID(ctx, fromUserID).Return(fromWallet, nil)
		walletRepo.EXPECT().FindByUserID(ctx, toUserID).Return(toWallet, nil)
		txRepo.EXPECT().FindByFromIDAndReference(ctx, fromWalletID, "tf-001").Return(nil, nil)
		withTxPassthrough(txManager)
		txRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		walletRepo.EXPECT().UpdateBalanceWithVersion(gomock.Any(), gomock.Any()).Return(nil) // lock sender
		walletRepo.EXPECT().UpdateBalanceWithVersion(gomock.Any(), gomock.Any()).Return(nil) // settle sender
		walletRepo.EXPECT().UpdateBalanceWithVersion(gomock.Any(), gomock.Any()).Return(nil) // credit receiver
		txRepo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), domain.StatusSuccess, 1).Return(nil)

		tx, updatedWallet, err := svc.Transfer(ctx, fromUserID, "bob", amount, "tf-001")
		assert.NoError(t, err)
		assert.Equal(t, domain.ActionTransfer, tx.Action)
		assert.Equal(t, decimal.NewFromFloat(900), updatedWallet.Balance)
	})

	t.Run("recipient not found", func(t *testing.T) {
		userRepo.EXPECT().FindByUsername(ctx, "ghost").Return(nil, domain.ErrUserNotFound)

		_, _, err := svc.Transfer(ctx, fromUserID, "ghost", decimal.NewFromFloat(100), "tf-002")
		assert.ErrorIs(t, err, domain.ErrRecipientNotFound)
	})

	t.Run("insufficient balance", func(t *testing.T) {
		fromWallet := &domain.Wallet{
			ID: fromWalletID, UserID: fromUserID,
			Balance: decimal.NewFromFloat(50), LockedAmount: decimal.Zero, Version: 1,
		}
		recipient := &domain.User{ID: toUserID, Username: "bob"}

		userRepo.EXPECT().FindByUsername(ctx, "bob").Return(recipient, nil)
		walletRepo.EXPECT().FindByUserID(ctx, fromUserID).Return(fromWallet, nil)

		_, _, err := svc.Transfer(ctx, fromUserID, "bob", decimal.NewFromFloat(100), "tf-003")
		assert.ErrorIs(t, err, domain.ErrInsufficientBalance)
	})
}
