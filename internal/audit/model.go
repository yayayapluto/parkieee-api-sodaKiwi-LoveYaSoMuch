package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     *uuid.UUID      `gorm:"type:uuid;index"`
	Action     string          `gorm:"type:varchar(100);not null"`
	EntityType string          `gorm:"type:varchar(50)"`
	EntityID   *uuid.UUID      `gorm:"type:uuid"`
	Metadata   json.RawMessage `gorm:"type:jsonb"`
	IPAddress  string          `gorm:"type:varchar(45)"`
	CreatedAt  time.Time       `gorm:"index"`
}

func AllModels() []any { return []any{&AuditLog{}} }
