package override

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo interface {
	Create(tx *gorm.DB, o *OperatorOverride) error
	CountTodayByOperator(operatorID uuid.UUID) (int64, error)
	FindActiveConfig() (*LocalConfig, error)
	FindTransaction(id uuid.UUID) (*LocalTx, error)
	UpdateTransactionFee(tx *gorm.DB, id uuid.UUID, fee int) error
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(tx *gorm.DB, o *OperatorOverride) error {
	o.CreatedAt = time.Now()
	return r.session(tx).Create(o).Error
}

func (r *Repository) CountTodayByOperator(operatorID uuid.UUID) (int64, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var count int64
	err := r.db.Model(&OperatorOverride{}).
		Where("operator_id = ? AND created_at >= ?", operatorID, today).
		Count(&count).Error
	return count, err
}

func (r *Repository) FindActiveConfig() (*LocalConfig, error) {
	var row struct {
		MaxOverridesPerDay     int        `gorm:"column:max_overrides_per_day"`
		EscalationNotifyUserID *uuid.UUID `gorm:"column:escalation_notify_user_id"`
	}
	err := r.db.Table("override_configs").
		Where("is_active = true").
		Order("created_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &LocalConfig{
		MaxOverridesPerDay:     row.MaxOverridesPerDay,
		EscalationNotifyUserID: row.EscalationNotifyUserID,
	}, nil
}

func (r *Repository) FindTransaction(id uuid.UUID) (*LocalTx, error) {
	var row struct {
		ID            uuid.UUID  `gorm:"column:id"`
		Status        string     `gorm:"column:status"`
		CalculatedFee int        `gorm:"column:calculated_fee"`
		ExitGateID    *uuid.UUID `gorm:"column:exit_gate_id"`
		ZoneID        uuid.UUID  `gorm:"column:zone_id"`
	}
	err := r.db.Table("transactions").Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &LocalTx{
		ID:            row.ID,
		Status:        row.Status,
		CalculatedFee: row.CalculatedFee,
		ExitGateID:    row.ExitGateID,
		ZoneID:        row.ZoneID,
	}, nil
}

func (r *Repository) UpdateTransactionFee(tx *gorm.DB, id uuid.UUID, fee int) error {
	return r.session(tx).Table("transactions").Where("id = ?", id).
		Updates(map[string]any{"calculated_fee": fee, "updated_at": time.Now()}).Error
}

func (r *Repository) session(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}
