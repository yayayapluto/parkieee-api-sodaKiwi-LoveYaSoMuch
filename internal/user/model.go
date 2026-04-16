package user

import (
	"time"

	"github.com/google/uuid"
)

// Role is a lightweight relation model for users.role_id preload.
// Table name remains `roles` to match auth ownership.
type Role struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name string    `gorm:"type:varchar(50)"`
}

// User owns the `users` table schema for the user module.
type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string     `gorm:"type:varchar(100);not null"`
	Username     string     `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email        string     `gorm:"type:varchar(150);uniqueIndex;not null"`
	PasswordHash string     `gorm:"type:text;not null"`
	RoleID       uuid.UUID  `gorm:"type:uuid;not null"`
	IsActive     bool       `gorm:"default:true"`
	TenantID     uuid.UUID  `gorm:"type:uuid"`
	CreatedBy    *uuid.UUID `gorm:"type:uuid"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time `gorm:"index"`

	Role Role `gorm:"foreignKey:RoleID;references:ID"`
}

// AllModels returns module-owned models for migration wiring.
func AllModels() []any {
	return []any{&User{}}
}
