package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port/mocks"
	"github.com/ingunawandra/mini-wallet/internal/core/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)
	txManager := mocks.NewMockTxManager(ctrl)
	tokenSvc := mocks.NewMockTokenService(ctrl)

	svc := service.NewAuthService(userRepo, walletRepo, txManager, tokenSvc, 4)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		txManager.EXPECT().
			WithTx(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})
		userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		walletRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		user, err := svc.Register(ctx, "alice", "password123")
		assert.NoError(t, err)
		assert.Equal(t, "alice", user.Username)
		assert.NotEqual(t, uuid.Nil, user.ID)
	})

	t.Run("duplicate username", func(t *testing.T) {
		txManager.EXPECT().
			WithTx(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})
		userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrUserAlreadyExists)

		user, err := svc.Register(ctx, "alice", "password123")
		assert.Nil(t, user)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	walletRepo := mocks.NewMockWalletRepository(ctrl)
	txManager := mocks.NewMockTxManager(ctrl)
	tokenSvc := mocks.NewMockTokenService(ctrl)

	svc := service.NewAuthService(userRepo, walletRepo, txManager, tokenSvc, 4)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 4)
		user := &domain.User{
			ID:             uuid.New(),
			Username:       "alice",
			HashedPassword: string(hash),
		}

		userRepo.EXPECT().FindByUsername(ctx, "alice").Return(user, nil)
		tokenSvc.EXPECT().
			Generate(user.ID, user.Username).
			Return("jwt-token", int64(9999999999), nil)

		tok, _, err := svc.Login(ctx, "alice", "password123")
		assert.NoError(t, err)
		assert.Equal(t, "jwt-token", tok)
	})

	t.Run("user not found returns invalid credentials", func(t *testing.T) {
		userRepo.EXPECT().FindByUsername(ctx, "ghost").Return(nil, domain.ErrUserNotFound)

		_, _, err := svc.Login(ctx, "ghost", "pass")
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("wrong password", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), 4)
		user := &domain.User{
			ID:             uuid.New(),
			Username:       "alice",
			HashedPassword: string(hash),
		}
		userRepo.EXPECT().FindByUsername(ctx, "alice").Return(user, nil)

		_, _, err := svc.Login(ctx, "alice", "wrongpassword")
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}
