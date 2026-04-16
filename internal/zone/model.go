package zone

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Zone struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name             string    `gorm:"type:varchar(100);not null"`
	Description      string    `gorm:"type:text"`
	Capacity         int
	AdditionalFee    int
	ForVehicleTypeID *uuid.UUID `gorm:"type:uuid"`
	IsActive         bool       `gorm:"default:true"`
	TenantID         uuid.UUID  `gorm:"type:uuid"`
	CreatedBy        *uuid.UUID `gorm:"type:uuid"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Gate struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ZoneID          uuid.UUID `gorm:"type:uuid;not null;index"`
	Name            string    `gorm:"type:varchar(100);not null"`
	GateType        string    `gorm:"type:varchar(10);not null"`
	Mode            string    `gorm:"type:varchar(20);not null"`
	LocationDesc    string    `gorm:"type:text"`
	GateToken       string    `gorm:"type:varchar(60);uniqueIndex"`
	TokenLastUsedAt *time.Time
	IsActive        bool       `gorm:"default:true"`
	TenantID        uuid.UUID  `gorm:"type:uuid"`
	CreatedBy       *uuid.UUID `gorm:"type:uuid"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type GateCashierAssignment struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	GateID     uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null"`
	AssignedBy *uuid.UUID `gorm:"type:uuid"`
	AssignedAt time.Time
}

type GateDevice struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	GateID       uuid.UUID `gorm:"type:uuid;not null;index"`
	DeviceType   string    `gorm:"type:varchar(30);not null"`
	Status       string    `gorm:"type:varchar(20);not null"`
	LastPingAt   *time.Time
	ErrorMessage string `gorm:"type:text"`
	UpdatedAt    time.Time
}

type GatePairingCode struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code        string    `gorm:"type:varchar(6);uniqueIndex;not null"`
	Status      string    `gorm:"type:varchar(20);not null"`
	ExpiresAt   time.Time
	CreatedAt   time.Time
	GateID      *uuid.UUID `gorm:"type:uuid"`
	ConfirmedBy *uuid.UUID `gorm:"type:uuid"`
	ConfirmedAt *time.Time
	GateJWT     string `gorm:"type:text"`
	IPAddress   string `gorm:"type:varchar(45)"`
}

const (
	PairingStatusPending   = "pending"
	PairingStatusConfirmed = "confirmed"
	PairingStatusExpired   = "expired"
)

func NewGateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(b), "=")
}

// AllModels returns module-owned models for migration wiring.
func AllModels() []any {
	return []any{
		&Zone{},
		&Gate{},
		&GateCashierAssignment{},
		&GateDevice{},
		&GatePairingCode{},
	}
}
