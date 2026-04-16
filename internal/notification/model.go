package notification

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    *uuid.UUID `gorm:"type:uuid;index"` // nil = broadcast
	Type      string     `gorm:"type:varchar(50);not null"`
	Title     string     `gorm:"type:varchar(200)"`
	Body      string     `gorm:"type:text"`
	IsRead    bool       `gorm:"default:false"`
	ReadAt    *time.Time
	CreatedAt time.Time `gorm:"index"`
}

func AllModels() []any { return []any{&Notification{}} }
