package vehicle

import (
	"time"

	"github.com/google/uuid"
)

type VehicleType struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(50);uniqueIndex;not null"`
	MinimumFee  int
	Description string `gorm:"type:text"`
	CreatedAt   time.Time
}

type Vehicle struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PlateNumber   string     `gorm:"type:varchar(20);index"`
	VehicleTypeID *uuid.UUID `gorm:"type:uuid"`
	Source        string     `gorm:"type:varchar(20)"`
	Notes         string     `gorm:"type:text"`
	TenantID      uuid.UUID  `gorm:"type:uuid"`
	CreatedAt     time.Time
	UpdatedAt     time.Time

	VehicleType *VehicleType `gorm:"foreignKey:VehicleTypeID;references:ID"`
}

type RFIDCard struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CardUID       string    `gorm:"type:varchar(64);uniqueIndex;not null"`
	VehicleID     uuid.UUID `gorm:"type:uuid;not null;index"`
	IsActive      bool      `gorm:"default:true"`
	CreatedAt     time.Time
	DeactivatedAt *time.Time
	DeactivatedBy *uuid.UUID `gorm:"type:uuid"`
}

// AllModels returns module-owned models for migration wiring.
func AllModels() []any {
	return []any{
		&VehicleType{},
		&Vehicle{},
		&RFIDCard{},
	}
}
