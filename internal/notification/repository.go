package notification

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo interface {
	Create(n *Notification) error
	FindByUserID(userID *uuid.UUID, page, limit int) ([]Notification, int64, error)
	CountUnread(userID *uuid.UUID) (int64, error)
	MarkRead(id uuid.UUID) error
	MarkAllRead(userID *uuid.UUID) error
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(n *Notification) error {
	n.CreatedAt = time.Now()
	return r.db.Create(n).Error
}

func (r *Repository) FindByUserID(userID *uuid.UUID, page, limit int) ([]Notification, int64, error) {
	var rows []Notification
	var total int64

	q := r.db.Model(&Notification{}).Where("user_id = ? OR user_id IS NULL", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) MarkRead(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&Notification{}).
		Where("id = ?", id).
		Updates(map[string]any{"is_read": true, "read_at": now}).Error
}

func (r *Repository) CountUnread(userID *uuid.UUID) (int64, error) {
	var total int64
	err := r.db.Model(&Notification{}).
		Where("is_read = false AND (user_id = ? OR user_id IS NULL)", userID).
		Count(&total).Error
	return total, err
}

func (r *Repository) MarkAllRead(userID *uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&Notification{}).
		Where("is_read = false AND user_id = ?", userID).
		Updates(map[string]any{"is_read": true, "read_at": now}).Error
}
