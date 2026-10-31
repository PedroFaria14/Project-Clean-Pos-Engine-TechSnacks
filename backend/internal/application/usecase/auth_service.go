package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("authentication required")
	ErrLoginLimited       = errors.New("too many login attempts")
	ErrAuthUnavailable    = errors.New("authentication temporarily unavailable")
)

const SessionLifetime = 8 * time.Hour

type AuthSession struct {
	User      identity.Principal `json:"user"`
	CSRFToken string             `json:"csrf_token"`
	ExpiresAt time.Time          `json:"expires_at"`
}

type AuthService struct {
	repo      port.AuthRepository
	hasher    port.PasswordHasher
	dummyHash string
	work      chan struct{}
}

func NewAuthService(repo port.AuthRepository, hasher port.PasswordHasher) (*AuthService, error) {
	dummy, err := hasher.Hash(rand.Text())
	if err != nil {
		return nil, err
	}
	return &AuthService{repo: repo, hasher: hasher, dummyHash: dummy, work: make(chan struct{}, 2)}, nil
}

func tokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func csrfToken(token string) string {
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write([]byte("techsnacks-csrf-v1"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func authUnavailable(err error) error { return fmt.Errorf("%w: %w", ErrAuthUnavailable, err) }

func (s *AuthService) Login(ctx context.Context, username, password, address string) (AuthSession, string, error) {
	normalized, valid := identity.NormalizeUsername(username)
	if !valid || len(password) < 1 || len(password) > 128 {
		return AuthSession{}, "", ErrInvalidCredentials
	}
	allowed, err := s.repo.AllowLogin(ctx, tokenHash("account:"+normalized), tokenHash("address:"+address))
	if err != nil {
		return AuthSession{}, "", authUnavailable(err)
	}
	if !allowed {
		return AuthSession{}, "", ErrLoginLimited
	}
	user, err := s.repo.FindUser(ctx, normalized)
	if err != nil && !errors.Is(err, port.ErrUserNotFound) {
		return AuthSession{}, "", authUnavailable(err)
	}
	found := err == nil && user.Active
	encoded := s.dummyHash
	if found {
		encoded = user.PasswordHash
	}
	select {
	case s.work <- struct{}{}:
		defer func() { <-s.work }()
	case <-ctx.Done():
		return AuthSession{}, "", authUnavailable(ctx.Err())
	}
	match, err := s.hasher.Verify(encoded, password)
	if err != nil {
		return AuthSession{}, "", authUnavailable(err)
	}
	if !found || !match {
		return AuthSession{}, "", ErrInvalidCredentials
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return AuthSession{}, "", authUnavailable(err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := time.Now().UTC().Add(SessionLifetime)
	if err := s.repo.CreateSession(ctx, tokenHash(token), user.ID, expires); err != nil {
		return AuthSession{}, "", authUnavailable(err)
	}
	return AuthSession{User: user.Principal, CSRFToken: csrfToken(token), ExpiresAt: expires}, token, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (AuthSession, error) {
	if len(token) != 43 {
		return AuthSession{}, ErrUnauthenticated
	}
	user, expires, err := s.repo.FindSession(ctx, tokenHash(token))
	if errors.Is(err, port.ErrSessionNotFound) {
		return AuthSession{}, ErrUnauthenticated
	}
	if err != nil {
		return AuthSession{}, authUnavailable(err)
	}
	return AuthSession{User: user, CSRFToken: csrfToken(token), ExpiresAt: expires}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if err := s.repo.DeleteSession(ctx, tokenHash(token)); err != nil {
		return authUnavailable(err)
	}
	return nil
}
