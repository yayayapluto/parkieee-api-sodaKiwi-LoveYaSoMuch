// internal/auth/repository.go
package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindUserByUsername(username string) (*User, error) {
	var u User
	err := r.db.Preload("Role").Where("username = ? AND deleted_at IS NULL", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *Repository) FindUserByID(id uuid.UUID) (*User, error) {
	var u User
	err := r.db.Preload("Role").Where("id = ? AND deleted_at IS NULL", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *Repository) FindPermissionsByRoleID(roleID uuid.UUID) ([]string, error) {
	var nodes []string
	err := r.db.Model(&Permission{}).
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Where("rp.role_id = ?", roleID).
		Pluck("permissions.node", &nodes).Error
	return nodes, err
}

func (r *Repository) CreateSession(sess *UserSession) error {
	return r.db.Create(sess).Error
}

func (r *Repository) CreateRefreshToken(rt *RefreshToken) error {
	return r.db.Create(rt).Error
}

func (r *Repository) FindRefreshToken(tokenHash string) (*RefreshToken, error) {
	var rt RefreshToken
	err := r.db.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now()).First(&rt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rt, err
}

func (r *Repository) RevokeRefreshToken(id uuid.UUID, replacedBy *uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&RefreshToken{}).Where("id = ?", id).
		Updates(map[string]any{"revoked_at": now, "replaced_by": replacedBy}).Error
}

func (r *Repository) RevokeSession(sessionID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&UserSession{}).Where("id = ?", sessionID).
		Update("revoked_at", now).Error
}

func (r *Repository) RevokeAllUserRefreshTokens(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (r *Repository) WriteLoginLog(log *UserLoginLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) UpsertLoginStats(userID uuid.UUID, success bool, ip string) error {
	now := time.Now()
	stats := UserLoginStats{UserID: userID, UpdatedAt: now}
	result := r.db.Where("user_id = ?", userID).First(&stats)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		stats.ID = uuid.New()
		stats.TotalAttempts = 1
		if !success {
			stats.TotalFailedAttempts = 1
			stats.LastFailedIP = ip
		} else {
			stats.LastSuccessAt = &now
		}
		stats.LastAttemptAt = &now
		return r.db.Create(&stats).Error
	}
	updates := map[string]any{
		"total_attempts": gorm.Expr("total_attempts + 1"),
		"last_attempt_at": now,
		"updated_at":      now,
	}
	if !success {
		updates["total_failed_attempts"] = gorm.Expr("total_failed_attempts + 1")
		updates["last_failed_ip"] = ip
	} else {
		updates["last_success_at"] = now
	}
	return r.db.Model(&UserLoginStats{}).Where("user_id = ?", userID).Updates(updates).Error
}
