package fee

import (
	"time"

	"github.com/google/uuid"
)

type FeeConfig struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ZoneID             uuid.UUID `gorm:"type:uuid;not null;index"`
	VehicleTypeID      uuid.UUID `gorm:"type:uuid;not null;index"`
	BaseFee            int
	GracePeriodMinutes int
	IsActive           bool `gorm:"default:true"`
	EffectiveFrom      time.Time
	EffectiveUntil     *time.Time
	TenantID           uuid.UUID  `gorm:"type:uuid"`
	CreatedBy          *uuid.UUID `gorm:"type:uuid"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type FeeTier struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	FeeConfigID     uuid.UUID `gorm:"type:uuid;not null;index"`
	TierOrder       int
	DurationMinutes int
	FeeAmount       int
	IsLastTier      bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type HolidayRate struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name          string    `gorm:"type:varchar(100);not null"`
	HolidayDate   time.Time `gorm:"type:date;not null;index"`
	AdditionalFee int
	Multiplier    float64 `gorm:"type:decimal(6,3);default:1"`
	IsActive      bool    `gorm:"default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OCRConfig struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AutoAcceptThreshold float64   `gorm:"type:decimal(5,4)"`
	IsActive            bool      `gorm:"default:true"`
	EffectiveFrom       time.Time
	CreatedBy           *uuid.UUID `gorm:"type:uuid"`
	CreatedAt           time.Time
}

type OverrideConfig struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MaxOverridesPerDay     int
	MaxOverridesPerWeek    int
	EscalationNotifyUserID *uuid.UUID `gorm:"type:uuid"`
	IsActive               bool       `gorm:"default:true"`
	CreatedBy              *uuid.UUID `gorm:"type:uuid"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type SystemConfig struct {
	Key       string `gorm:"type:varchar(100);primaryKey"`
	Value     string `gorm:"type:text"`
	UpdatedAt time.Time
}

// AllModels returns module-owned models for migration wiring.
func AllModels() []any {
	return []any{
		&FeeConfig{},
		&FeeTier{},
		&HolidayRate{},
		&OCRConfig{},
		&OverrideConfig{},
		&SystemConfig{},
	}
}
