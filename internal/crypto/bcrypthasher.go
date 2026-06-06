// internal/crypto/password.go
package crypto

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const DefaultCost = 12

type PasswordHasher interface {
	Hash(s string) (string, error)
	CompareHashAndPassword(hash, password string) (bool, error)
}

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) *BcryptHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

func (h *BcryptHasher) Hash(s string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *BcryptHasher) CompareHashAndPassword(hash, plain string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return false, err
}
