package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
)

type AuthHandler struct {
	authService port.AuthService
}

func NewAuthHandler(authService port.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50,alphanum"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		domainErrToHTTP(c, err)
		return
	}

	success(c, http.StatusCreated, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"created_at": user.CreatedAt.Format(time.RFC3339),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	token, expiresAt, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		domainErrToHTTP(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{
		"access_token": token,
		"expires_at":   time.Unix(expiresAt, 0).UTC().Format(time.RFC3339),
	})
}
