package payment

import (
	"time"

	"github.com/google/uuid"
)

const (
	MethodCash = "cash"
	MethodQRIS = "qris"
)

const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

type Payment struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID         uuid.UUID `gorm:"type:uuid;not null;index"`
	Method                string    `gorm:"type:varchar(20);not null"`
	Amount                int
	Status                string     `gorm:"type:varchar(20);not null"`
	HandledByUserID       *uuid.UUID `gorm:"type:uuid"`
	CashTendered          int
	CashChange            int
	MidtransOrderID       string `gorm:"type:varchar(100);uniqueIndex"`
	MidtransTransactionID string `gorm:"type:varchar(100)"`
	QRISString            string `gorm:"type:text"`
	QRISImageURL          string `gorm:"type:text"`
	QRISExpiresAt         *time.Time
	MidtransStatus        string `gorm:"type:varchar(50)"`
	PaidAt                *time.Time
	TenantID              uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type MidtransCallback struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PaymentID       *uuid.UUID `gorm:"type:uuid"`
	MidtransOrderID string     `gorm:"type:varchar(100);index"`
	RawPayload      []byte     `gorm:"type:jsonb"`
	SignatureValid  bool
	Processed       bool
	ProcessedAt     *time.Time
	ReceivedAt      time.Time
	ErrorMessage    string `gorm:"type:text"`
}

const (
	RefundStatusApproved = "approved"
)

type Refund struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID uuid.UUID `gorm:"type:uuid;not null;index"`
	PaymentID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Amount        int
	Reason        string `gorm:"type:text"`
	Status        string `gorm:"type:varchar(20);not null"`
	RequestedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func AllModels() []any {
	return []any{&Payment{}, &MidtransCallback{}, &Refund{}}
}
