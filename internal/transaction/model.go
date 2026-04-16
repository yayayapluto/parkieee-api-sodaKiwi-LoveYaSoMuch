package transaction

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive         = "active"
	StatusPendingPayment = "pending_payment"
	StatusCompleted      = "completed"
	StatusCancelled      = "cancelled"
)

const (
	MethodRFID   = "rfid"
	MethodTicket = "ticket"
)

type Transaction struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionCode    string     `gorm:"type:varchar(50);uniqueIndex;not null"`
	EntryGateID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	EntryMethod        string     `gorm:"type:varchar(10);not null"`
	RFIDCardID         *uuid.UUID `gorm:"type:uuid"`
	TicketCode         *string    `gorm:"type:varchar(255);uniqueIndex"`
	TicketCodeImage    string     `gorm:"type:text"`
	EntryAt            time.Time
	EntryPhotoURL      string     `gorm:"type:text"`
	EntryPhotoPath     string     `gorm:"type:text"`
	ExitGateID         *uuid.UUID `gorm:"type:uuid"`
	ExitMethod         *string    `gorm:"type:varchar(10)"`
	ExitAt             *time.Time
	ExitPhotoURL       string     `gorm:"type:text"`
	ExitPhotoPath      string     `gorm:"type:text"`
	VehicleID          *uuid.UUID `gorm:"type:uuid;index"`
	FeeConfigID        *uuid.UUID `gorm:"type:uuid"`
	CalculatedFee      int
	ReceiptPrinted     bool `gorm:"default:false"`
	ReceiptPrintedAt   *time.Time
	PlateMismatch      bool `gorm:"default:false"`
	CashierRequestedAt *time.Time
	Status             string    `gorm:"type:varchar(20);not null;default:'active'"`
	ZoneID             uuid.UUID `gorm:"type:uuid;not null;index"`
	TenantID           uuid.UUID `gorm:"type:uuid"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type TransactionLog struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	FromStatus        string     `gorm:"type:varchar(20)"`
	ToStatus          string     `gorm:"type:varchar(20)"`
	Event             string     `gorm:"type:varchar(50)"`
	TriggeredBy       string     `gorm:"type:varchar(20)"`
	TriggeredByUserID *uuid.UUID `gorm:"type:uuid"`
	Note              string     `gorm:"type:text"`
	Metadata          []byte     `gorm:"type:jsonb"`
	CreatedAt         time.Time
}

type UnclosedTransactionFlag struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID  uuid.UUID `gorm:"type:uuid;not null;index"`
	FlaggedAt      time.Time
	FlagType       string `gorm:"type:varchar(30)"`
	FlagReason     string `gorm:"type:varchar(100)"`
	Resolved       bool   `gorm:"default:false"`
	ResolvedAt     *time.Time
	ResolvedBy     *uuid.UUID `gorm:"type:uuid"`
	ResolutionNote string     `gorm:"type:text"`
}

func AllModels() []any {
	return []any{&Transaction{}, &TransactionLog{}, &UnclosedTransactionFlag{}}
}
