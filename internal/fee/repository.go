package fee

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo interface {
	ListFeeConfigs() ([]FeeConfig, error)
	CreateFeeConfig(cfg *FeeConfig) error
	FindFeeConfigByID(id uuid.UUID) (*FeeConfig, error)
	UpdateFeeConfig(cfg *FeeConfig) error
	DeleteFeeConfig(id uuid.UUID) error

	ListFeeTiers(feeConfigID uuid.UUID) ([]FeeTier, error)
	CreateFeeTier(tier *FeeTier) error
	FindFeeTierByID(id uuid.UUID) (*FeeTier, error)
	UpdateFeeTier(tier *FeeTier) error
	DeleteFeeTier(id uuid.UUID) error

	ListHolidayRates() ([]HolidayRate, error)
	CreateHolidayRate(rate *HolidayRate) error
	FindHolidayRateByID(id uuid.UUID) (*HolidayRate, error)
	UpdateHolidayRate(rate *HolidayRate) error
	DeleteHolidayRate(id uuid.UUID) error

	ListSystemConfigs() ([]SystemConfig, error)
	UpdateSystemConfig(key, value string) error
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListFeeConfigs() ([]FeeConfig, error) {
	var rows []FeeConfig
	err := r.db.Order("effective_from DESC").Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateFeeConfig(cfg *FeeConfig) error {
	return r.db.Create(cfg).Error
}

func (r *Repository) FindFeeConfigByID(id uuid.UUID) (*FeeConfig, error) {
	var row FeeConfig
	err := r.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *Repository) UpdateFeeConfig(cfg *FeeConfig) error {
	cfg.UpdatedAt = time.Now()
	return r.db.Save(cfg).Error
}

func (r *Repository) DeleteFeeConfig(id uuid.UUID) error {
	return r.db.Delete(&FeeConfig{}, "id = ?", id).Error
}

func (r *Repository) ListFeeTiers(feeConfigID uuid.UUID) ([]FeeTier, error) {
	var rows []FeeTier
	err := r.db.Where("fee_config_id = ?", feeConfigID).Order("tier_order ASC").Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateFeeTier(tier *FeeTier) error {
	return r.db.Create(tier).Error
}

func (r *Repository) FindFeeTierByID(id uuid.UUID) (*FeeTier, error) {
	var row FeeTier
	err := r.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *Repository) UpdateFeeTier(tier *FeeTier) error {
	tier.UpdatedAt = time.Now()
	return r.db.Save(tier).Error
}

func (r *Repository) DeleteFeeTier(id uuid.UUID) error {
	return r.db.Delete(&FeeTier{}, "id = ?", id).Error
}

func (r *Repository) ListHolidayRates() ([]HolidayRate, error) {
	var rows []HolidayRate
	err := r.db.Order("holiday_date DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateHolidayRate(rate *HolidayRate) error {
	return r.db.Create(rate).Error
}

func (r *Repository) FindHolidayRateByID(id uuid.UUID) (*HolidayRate, error) {
	var row HolidayRate
	err := r.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *Repository) UpdateHolidayRate(rate *HolidayRate) error {
	rate.UpdatedAt = time.Now()
	return r.db.Save(rate).Error
}

func (r *Repository) DeleteHolidayRate(id uuid.UUID) error {
	return r.db.Delete(&HolidayRate{}, "id = ?", id).Error
}

func (r *Repository) ListSystemConfigs() ([]SystemConfig, error) {
	var rows []SystemConfig
	return rows, r.db.Order("key ASC").Find(&rows).Error
}

func (r *Repository) UpdateSystemConfig(key, value string) error {
	return r.db.Model(&SystemConfig{}).Where("key = ?", key).
		Updates(map[string]any{"value": value, "updated_at": time.Now()}).Error
}
