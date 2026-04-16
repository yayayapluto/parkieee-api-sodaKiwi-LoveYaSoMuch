package ocr

import (
	"time"

	"github.com/google/uuid"
)

const (
	PhotoTypeEntry = "entry"
	PhotoTypeExit  = "exit"
)

const (
	JobStatusQueued     = "queued"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

type OCRJob struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID   uuid.UUID `gorm:"type:uuid;not null;index"`
	ImageURL        string    `gorm:"type:text;not null"`
	PhotoType       string    `gorm:"type:varchar(10);not null"`
	Status          string    `gorm:"type:varchar(20);not null"`
	RetryCount      int
	QueuedAt        time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	ErrorMessage    string `gorm:"type:text"`
	OutputImagePath string `gorm:"type:text"`
	OutputImageURL  string `gorm:"type:text"`
}

type OCRResult struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OCRJobID      uuid.UUID `gorm:"type:uuid;not null;index"`
	PlateDetected string    `gorm:"type:varchar(30)"`
	ActualPlate   string    `gorm:"type:varchar(30)"`
	IsMatch       bool
	Confidence    float64    `gorm:"type:decimal(5,4)"`
	RawOutput     []byte     `gorm:"type:jsonb"`
	VehicleID     *uuid.UUID `gorm:"type:uuid"`
	IsVerified    bool       `gorm:"default:false"`
	VerifiedBy    *uuid.UUID `gorm:"type:uuid"`
	VerifiedAt    *time.Time
	CreatedAt     time.Time
}

type OCRReviewLog struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OCRResultID     uuid.UUID `gorm:"type:uuid;not null;index"`
	OCRPlate        string    `gorm:"type:varchar(30)"`
	OCRConfidence   float64   `gorm:"type:decimal(5,4)"`
	ManualPlate     string    `gorm:"type:varchar(30)"`
	ReviewedBy      uuid.UUID `gorm:"type:uuid;not null"`
	Match           bool
	AutoMatchResult bool
	ReviewNote      string `gorm:"type:text"`
	ReviewedAt      time.Time
	CreatedAt       time.Time
}

func AllModels() []any {
	return []any{&OCRJob{}, &OCRResult{}, &OCRReviewLog{}}
}
