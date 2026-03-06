package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ingunawandra/mini-wallet/internal/core/domain"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
	"github.com/shopspring/decimal"
)

const userIDKey = "user_id"

type WalletHandler struct {
	walletService port.WalletService
}

func NewWalletHandler(walletService port.WalletService) *WalletHandler {
	return &WalletHandler{walletService: walletService}
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID := mustUserID(c)

	wallet, err := h.walletService.GetBalance(c.Request.Context(), userID)
	if err != nil {
		domainErrToHTTP(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{
		"balance":           wallet.Balance.StringFixed(2),
		"locked_amount":     wallet.LockedAmount.StringFixed(2),
		"available_balance": wallet.AvailableBalance().StringFixed(2),
	})
}

type topupRequest struct {
	Amount      string `json:"amount" binding:"required"`
	ReferenceID string `json:"reference_id" binding:"required,max=100"`
}

func (h *WalletHandler) TopUp(c *gin.Context) {
	var req topupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	amount, err := parsePositiveDecimal(req.Amount)
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_REQUEST", "amount must be a positive number")
		return
	}

	userID := mustUserID(c)
	tx, wallet, err := h.walletService.TopUp(c.Request.Context(), userID, amount, req.ReferenceID)
	if err == domain.ErrDuplicateReference && tx != nil {
		success(c, http.StatusCreated, gin.H{
			"transaction_id": tx.ID,
			"balance":        wallet.Balance.StringFixed(2),
		})
		return
	}
	if err != nil {
		domainErrToHTTP(c, err)
		return
	}

	success(c, http.StatusCreated, gin.H{
		"transaction_id": tx.ID,
		"balance":        wallet.Balance.StringFixed(2),
	})
}

type withdrawRequest struct {
	Amount      string `json:"amount" binding:"required"`
	ReferenceID string `json:"reference_id" binding:"required,max=100"`
}

func (h *WalletHandler) Withdraw(c *gin.Context) {
	var req withdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	amount, err := parsePositiveDecimal(req.Amount)
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_REQUEST", "amount must be a positive number")
		return
	}

	userID := mustUserID(c)
	tx, wallet, err := h.walletService.Withdraw(c.Request.Context(), userID, amount, req.ReferenceID)
	if err == domain.ErrDuplicateReference && tx != nil {
		success(c, http.StatusCreated, gin.H{
			"transaction_id":    tx.ID,
			"available_balance": wallet.AvailableBalance().StringFixed(2),
		})
		return
	}
	if err != nil {
		domainErrToHTTP(c, err)
		return
	}

	success(c, http.StatusCreated, gin.H{
		"transaction_id":    tx.ID,
		"available_balance": wallet.AvailableBalance().StringFixed(2),
	})
}

type transferRequest struct {
	ToUsername  string `json:"to_username" binding:"required"`
	Amount      string `json:"amount" binding:"required"`
	ReferenceID string `json:"reference_id" binding:"required,max=100"`
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	var req transferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	amount, err := parsePositiveDecimal(req.Amount)
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_REQUEST", "amount must be a positive number")
		return
	}

	userID := mustUserID(c)
	tx, wallet, err := h.walletService.Transfer(c.Request.Context(), userID, req.ToUsername, amount, req.ReferenceID)
	if err == domain.ErrDuplicateReference && tx != nil {
		success(c, http.StatusCreated, gin.H{
			"transaction_id":    tx.ID,
			"available_balance": wallet.AvailableBalance().StringFixed(2),
		})
		return
	}
	if err != nil {
		domainErrToHTTP(c, err)
		return
	}

	success(c, http.StatusCreated, gin.H{
		"transaction_id":    tx.ID,
		"available_balance": wallet.AvailableBalance().StringFixed(2),
	})
}

func mustUserID(c *gin.Context) uuid.UUID {
	return c.MustGet(userIDKey).(uuid.UUID)
}

func parsePositiveDecimal(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil || !d.IsPositive() {
		return decimal.Zero, err
	}
	return d, nil
}
