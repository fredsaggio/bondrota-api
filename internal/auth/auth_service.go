package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

const TokenTTL = 24 * time.Hour

type Claims struct {
	jwt.RegisteredClaims
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

type AuthService struct {
	hasher    crypto.PasswordHasher
	jwtSecret []byte
}

func NewAuthService(hasher crypto.PasswordHasher, jwtSecret string) *AuthService {
	return &AuthService{
		hasher:    hasher,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *AuthService) GenerateToken(userID int64, role string) (string, error) {
	const op = "auth/AuthService.GenerateToken"

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
		Role:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return signed, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	const op = "auth/AuthService.ValidateToken"

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("%s: invalid token", op)
	}

	return claims, nil
}

func (s *AuthService) HashPassword(plain string) (string, error) {
	return s.hasher.Hash(plain)
}

func (s *AuthService) CheckPassword(hash, plain string) (bool, error) {
	return s.hasher.CompareHashAndPassword(hash, plain)
}
