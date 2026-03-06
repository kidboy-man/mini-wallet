package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo     port.UserRepository
	walletRepo   port.WalletRepository
	txManager    port.TxManager
	tokenService port.TokenService
	bcryptCost   int
}

// NewAuthService creates an AuthService.
func NewAuthService(
	userRepo port.UserRepository,
	walletRepo port.WalletRepository,
	txManager port.TxManager,
	tokenService port.TokenService,
	bcryptCost int,
) port.AuthService {
	return &authService{
		userRepo:     userRepo,
		walletRepo:   walletRepo,
		txManager:    txManager,
		tokenService: tokenService,
		bcryptCost:   bcryptCost,
	}
}

// Register creates a new user and their wallet atomically.
func (s *authService) Register(ctx context.Context, username, password string) (*domain.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return nil, domain.ErrInternalServer(err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:             uuid.New(),
		Username:       username,
		HashedPassword: string(hashed),
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	wallet := &domain.Wallet{
		ID:           uuid.New(),
		UserID:       user.ID,
		Balance:      decimal.Zero,
		LockedAmount: decimal.Zero,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.userRepo.Create(txCtx, user); err != nil {
			return err
		}
		return s.walletRepo.Create(txCtx, wallet)
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Login verifies credentials and returns a JWT token.
func (s *authService) Login(ctx context.Context, username, password string) (string, int64, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", 0, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		return "", 0, domain.ErrInvalidCredentials
	}

	return s.tokenService.Generate(user.ID, user.Username)
}
