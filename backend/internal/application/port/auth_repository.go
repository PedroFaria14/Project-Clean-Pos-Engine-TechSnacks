package port

import (
	"context"
	"errors"
	"time"

	"PedroFaria14/clean-pos-engine/internal/domain/identity"
)

var (
	ErrUserNotFound    = errors.New("account not found")
	ErrUserExists      = errors.New("account already exists")
	ErrSessionNotFound = errors.New("session not found")
)

type AuthRepository interface {
	FindUser(context.Context, string) (identity.User, error)
	CreateUser(context.Context, identity.User) error
	CreateSession(context.Context, string, string, time.Time) error
	FindSession(context.Context, string) (identity.Principal, time.Time, error)
	DeleteSession(context.Context, string) error
	AllowLogin(context.Context, string, string) (bool, error)
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Verify(string, string) (bool, error)
}
