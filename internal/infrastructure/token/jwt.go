package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ingunawandra/mini-wallet/internal/core/port"
)

type jwtClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secret        []byte
	expiryMinutes int
}

// NewJWTService creates a TokenService backed by HMAC-SHA256.
func NewJWTService(secret string, expiryMinutes int) port.TokenService {
	return &jwtService{
		secret:        []byte(secret),
		expiryMinutes: expiryMinutes,
	}
}

func (s *jwtService) Generate(userID uuid.UUID, username string) (string, int64, error) {
	expiresAt := time.Now().Add(time.Duration(s.expiryMinutes) * time.Minute)
	claims := jwtClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}

	return signed, expiresAt.Unix(), nil
}

func (s *jwtService) Validate(tokenStr string) (*port.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("invalid subject in token: %w", err)
	}

	return &port.TokenClaims{
		UserID:   userID,
		Username: claims.Username,
	}, nil
}
