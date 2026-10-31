package repository

import (
	"context"
	"errors"
	"time"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type UserModel struct {
	ID           string `gorm:"primaryKey"`
	Username     string
	PasswordHash string
	Role         string
	Active       bool
}

func (UserModel) TableName() string { return "users" }

type GormAuthRepository struct{ db *gorm.DB }

func NewGormAuthRepository(db *gorm.DB) *GormAuthRepository {
	// Authentication queries must never print account names, hashes or session material.
	return &GormAuthRepository{db: db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})}
}

func (r *GormAuthRepository) FindUser(ctx context.Context, username string) (identity.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("username = ?", username).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return identity.User{}, port.ErrUserNotFound
	}
	return identity.User{Principal: identity.Principal{ID: model.ID, Username: model.Username, Role: identity.Role(model.Role)}, PasswordHash: model.PasswordHash, Active: model.Active}, err
}

func (r *GormAuthRepository) CreateUser(ctx context.Context, user identity.User) error {
	model := UserModel{ID: user.ID, Username: user.Username, Role: string(user.Role), PasswordHash: user.PasswordHash, Active: user.Active}
	err := r.db.WithContext(ctx).Create(&model).Error
	if uniqueViolation(err) {
		return port.ErrUserExists
	}
	return err
}

func (r *GormAuthRepository) CreateSession(ctx context.Context, hash, userID string, expires time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM user_sessions WHERE expires_at <= NOW()").Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO user_sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)", hash, userID, expires).Error
	})
}

func (r *GormAuthRepository) FindSession(ctx context.Context, hash string) (identity.Principal, time.Time, error) {
	var result struct {
		ID        string
		Username  string
		Role      string
		ExpiresAt time.Time
	}
	query := r.db.WithContext(ctx).Raw(`SELECT u.id, u.username, u.role, s.expires_at FROM user_sessions s
		JOIN users u ON u.id = s.user_id WHERE s.token_hash = ? AND s.expires_at > NOW() AND u.active`, hash).Scan(&result)
	if query.Error != nil {
		return identity.Principal{}, time.Time{}, query.Error
	}
	if query.RowsAffected != 1 {
		return identity.Principal{}, time.Time{}, port.ErrSessionNotFound
	}
	return identity.Principal{ID: result.ID, Username: result.Username, Role: identity.Role(result.Role)}, result.ExpiresAt, nil
}

func (r *GormAuthRepository) DeleteSession(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM user_sessions WHERE token_hash = ?", hash).Error
}

func (r *GormAuthRepository) AllowLogin(ctx context.Context, account, address string) (bool, error) {
	allowed := true
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM auth_login_limits WHERE window_start < NOW() - interval '1 day'").Error; err != nil {
			return err
		}
		// Connection address is consumed first, bounding arbitrary-account bucket creation.
		for _, bucket := range []struct {
			key   string
			limit int
		}{{address, 100}, {account, 10}} {
			var attempts int
			if err := tx.Raw(`INSERT INTO auth_login_limits (bucket_key, attempts, window_start) VALUES (?, 1, NOW())
				ON CONFLICT (bucket_key) DO UPDATE SET
				attempts = CASE WHEN auth_login_limits.window_start <= NOW() - interval '15 minutes' THEN 1 ELSE LEAST(auth_login_limits.attempts + 1, 1000000) END,
				window_start = CASE WHEN auth_login_limits.window_start <= NOW() - interval '15 minutes' THEN NOW() ELSE auth_login_limits.window_start END
				RETURNING attempts`, bucket.key).Scan(&attempts).Error; err != nil {
				return err
			}
			if attempts > bucket.limit {
				allowed = false
				break
			}
		}
		return nil
	})
	return allowed, err
}
