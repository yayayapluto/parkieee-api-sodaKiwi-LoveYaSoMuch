package override

import (
	"time"

	"github.com/google/uuid"
)

type OperatorOverride struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID uuid.UUID `gorm:"type:uuid;not null;index"`
	OperatorID    uuid.UUID `gorm:"type:uuid;not null;index"`
	OverrideType  string    `gorm:"type:varchar(30);not null"` // "fee_adjust" | "barrier_release"
	OriginalFee   int
	AdjustedFee   int
	Reason        string    `gorm:"type:text"`
	CreatedAt     time.Time
}

// LocalTx is a minimal transaction snapshot — avoids importing internal/transaction.
type LocalTx struct {
	ID            uuid.UUID
	Status        string
	CalculatedFee int
	ExitGateID    *uuid.UUID
	ZoneID        uuid.UUID
}

// LocalConfig is a minimal override config snapshot — avoids importing internal/fee.
type LocalConfig struct {
	MaxOverridesPerDay     int
	EscalationNotifyUserID *uuid.UUID
}

func AllModels() []any { return []any{&OperatorOverride{}} }
