package user

import (
	"errors"
	"strings"
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

func (r *Repository) ListUsers(filter ListFilter) ([]User, int64, error) {
	q := r.db.Model(&User{}).Where("deleted_at IS NULL")

	if filter.RoleName != "" {
		q = q.Joins("JOIN roles ON roles.id = users.role_id").Where("roles.name = ?", filter.RoleName)
	}
	if filter.IsActive != nil {
		q = q.Where("users.is_active = ?", *filter.IsActive)
	}
	if filter.Search != "" {
		like := "%" + strings.ToLower(filter.Search) + "%"
		q = q.Where("LOWER(users.name) LIKE ? OR LOWER(users.username) LIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []User
	err := q.Preload("Role").
		Order("users.created_at DESC").
		Offset((filter.Page - 1) * filter.Limit).
		Limit(filter.Limit).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *Repository) FindUserByID(id uuid.UUID) (*User, error) {
	var u User
	err := r.db.Preload("Role").Where("id = ? AND deleted_at IS NULL", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *Repository) FindUserByUsername(username string) (*User, error) {
	var u User
	err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *Repository) CreateUser(u *User) error {
	return r.db.Create(u).Error
}

func (r *Repository) UpdateUser(u *User) error {
	u.UpdatedAt = time.Now()
	return r.db.Save(u).Error
}

func (r *Repository) SetUserActive(id uuid.UUID, isActive bool) error {
	return r.db.Model(&User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{"is_active": isActive, "updated_at": time.Now()}).Error
}

func (r *Repository) UpdatePassword(id uuid.UUID, passwordHash string) error {
	return r.db.Model(&User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{"password_hash": passwordHash, "updated_at": time.Now()}).Error
}

func (r *Repository) RoleExists(roleID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&Role{}).Where("id = ?", roleID).Count(&count).Error
	return count > 0, err
}
