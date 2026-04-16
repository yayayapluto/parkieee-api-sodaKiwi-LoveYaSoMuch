package ocr

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/internal/fee"
	"gorm.io/gorm"
)

type TransactionContext struct {
	TransactionID uuid.UUID
	VehicleID     *uuid.UUID
	ActualPlate   string
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateJob(tx *gorm.DB, job *OCRJob) error {
	if tx != nil {
		return tx.Create(job).Error
	}
	return r.db.Create(job).Error
}

func (r *Repository) UpdateJob(job *OCRJob) error {
	return r.db.Save(job).Error
}

func (r *Repository) CreateResult(result *OCRResult) error {
	return r.db.Create(result).Error
}

func (r *Repository) FindJobByID(id uuid.UUID) (*OCRJob, error) {
	var job OCRJob
	err := r.db.Where("id = ?", id).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &job, err
}

func (r *Repository) FindResultByJobID(jobID uuid.UUID) (*OCRResult, error) {
	var result OCRResult
	err := r.db.Where("ocr_job_id = ?", jobID).Order("created_at DESC").First(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &result, err
}

func (r *Repository) CreateReviewLog(log *OCRReviewLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) UpdateResult(result *OCRResult) error {
	return r.db.Save(result).Error
}

func (r *Repository) UpdateTransactionMismatch(transactionID uuid.UUID, mismatch bool) error {
	return r.db.Table("transactions").Where("id = ?", transactionID).Update("plate_mismatch", mismatch).Error
}

func (r *Repository) FindTransactionContext(transactionID uuid.UUID) (*TransactionContext, error) {
	var out TransactionContext
	err := r.db.
		Table("transactions t").
		Select("t.id AS transaction_id, t.vehicle_id, COALESCE(v.plate_number, '') AS actual_plate").
		Joins("LEFT JOIN vehicles v ON v.id = t.vehicle_id").
		Where("t.id = ?", transactionID).
		First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &out, err
}

func (r *Repository) FindActiveThreshold() (float64, error) {
	var cfg fee.OCRConfig
	now := time.Now()
	err := r.db.Where("is_active = ? AND effective_from <= ?", true, now).
		Order("effective_from DESC").
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return cfg.AutoAcceptThreshold, err
}

func (r *Repository) ListJobs(status string, page, limit int) ([]OCRJob, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	q := r.db.Model(&OCRJob{})
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []OCRJob
	err := q.Order("queued_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
