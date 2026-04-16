// internal/auth/model.go
package auth

import (
	"time"

	"github.com/google/uuid"
	usermodule "github.com/yyypluto/parkieee-api/internal/user"
)

type Role struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string     `gorm:"type:varchar(50);uniqueIndex;not null"`
	Description string     `gorm:"type:text"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Node        string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time
}

type RolePermission struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoleID       uuid.UUID `gorm:"type:uuid;not null;index"`
	PermissionID uuid.UUID `gorm:"type:uuid;not null"`
	GrantedBy    uuid.UUID `gorm:"type:uuid"`
	GrantedAt    time.Time
}

// User model ownership is in internal/user; auth consumes it as an alias.
type User = usermodule.User

type UserSession struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"type:text;not null"`
	IPAddress string    `gorm:"type:varchar(45)"`
	UserAgent string    `gorm:"type:text"`
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type RefreshToken struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	SessionID  uuid.UUID `gorm:"type:uuid;not null"`
	TokenHash  string    `gorm:"type:text;uniqueIndex;not null"`
	IPAddress  string    `gorm:"type:varchar(45)"`
	UserAgent  string    `gorm:"type:text"`
	CreatedAt  time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID `gorm:"type:uuid"`
}

type UserLoginLog struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID        uuid.UUID `gorm:"type:uuid;index"`
	IPAddress     string    `gorm:"type:varchar(45)"`
	UserAgent     string    `gorm:"type:text"`
	AttemptType   string    `gorm:"type:varchar(10)"`
	Success       bool
	FailureReason string `gorm:"type:varchar(100)"`
	AttemptedAt   time.Time
}

type UserLoginStats struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID              uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	TotalAttempts       int       `gorm:"default:0"`
	TotalFailedAttempts int       `gorm:"default:0"`
	LastAttemptAt       *time.Time
	LastSuccessAt       *time.Time
	LastFailedIP        string `gorm:"type:varchar(45)"`
	IsLocked            bool   `gorm:"default:false"`
	LockedAt            *time.Time
	LockedReason        string `gorm:"type:varchar(100)"`
	UpdatedAt           time.Time
}

// AllModels returns auth-owned models for AutoMigrate.
func AllModels() []any {
	return []any{
		&Role{}, &Permission{}, &RolePermission{},
		&UserSession{}, &RefreshToken{},
		&UserLoginLog{}, &UserLoginStats{},
	}
}
