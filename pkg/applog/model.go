package applog

import (
	"time"

	"github.com/google/uuid"
)

// AppLog stores structured slog records for queryable operational logs.
type AppLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Level     string    `gorm:"type:varchar(10);not null"`
	Source    string    `gorm:"type:varchar(200);not null;default:''"`
	Message   string    `gorm:"type:text;not null"`
	Fields    []byte    `gorm:"type:jsonb;not null"`
	RequestID *string   `gorm:"type:varchar(36)"`
	CreatedAt time.Time `gorm:"index"`
}

func (AppLog) TableName() string {
	return "app_logs"
}

func AllModels() []any {
	return []any{&AppLog{}}
}
