package transaction

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/internal/fee"
	"gorm.io/gorm"
)

type RFIDCard struct {
	ID            uuid.UUID
	CardUID       string
	VehicleID     uuid.UUID
	VehicleTypeID *uuid.UUID
	PlateNumber   string
	IsActive      bool
}

type Gate struct {
	ID       uuid.UUID
	ZoneID   uuid.UUID
	GateType string
	IsActive bool
}

type Vehicle struct {
	ID            uuid.UUID
	VehicleTypeID *uuid.UUID
	PlateNumber   string
}

type ListFilter struct {
	Page      int
	Limit     int
	Status    string
	ZoneID    *uuid.UUID
	StartDate *time.Time
	EndDate   *time.Time
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateTransaction(tx *gorm.DB, t *Transaction) error {
	return r.session(tx).Create(t).Error
}

func (r *Repository) FindActiveByVehicle(vehicleID uuid.UUID) (*Transaction, error) {
	var t Transaction
	err := r.db.Where("vehicle_id = ? AND status = ?", vehicleID, StatusActive).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (r *Repository) FindActiveByTicket(ticketCode string) (*Transaction, error) {
	var t Transaction
	err := r.db.Where("ticket_code = ? AND status = ?", ticketCode, StatusActive).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (r *Repository) FindByID(id uuid.UUID) (*Transaction, error) {
	var t Transaction
	err := r.db.Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (r *Repository) UpdateTransaction(tx *gorm.DB, t *Transaction) error {
	return r.session(tx).Save(t).Error
}

func (r *Repository) CreateLog(tx *gorm.DB, l *TransactionLog) error {
	return r.session(tx).Create(l).Error
}

func (r *Repository) FindRFIDCard(cardUID string) (*RFIDCard, error) {
	var out RFIDCard
	err := r.db.
		Table("rfid_cards AS r").
		Select("r.id, r.card_uid, r.vehicle_id, r.is_active, v.vehicle_type_id, v.plate_number").
		Joins("JOIN vehicles v ON v.id = r.vehicle_id").
		Where("r.card_uid = ? AND r.is_active = ?", cardUID, true).
		First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &out, err
}

func (r *Repository) FindGate(id uuid.UUID) (*Gate, error) {
	var out Gate
	err := r.db.Table("gates").Select("id, zone_id, gate_type, is_active").Where("id = ?", id).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &out, err
}

func (r *Repository) FindVehicleByID(id uuid.UUID) (*Vehicle, error) {
	var out Vehicle
	err := r.db.Table("vehicles").Select("id, vehicle_type_id, plate_number").Where("id = ?", id).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &out, err
}

func (r *Repository) FindActiveFeeConfig(zoneID, vehicleTypeID uuid.UUID) (*fee.FeeConfig, error) {
	var cfg fee.FeeConfig
	now := time.Now()
	err := r.db.Where(
		"zone_id = ? AND vehicle_type_id = ? AND is_active = ? AND effective_from <= ? AND (effective_until IS NULL OR effective_until >= ?)",
		zoneID, vehicleTypeID, true, now, now,
	).Order("effective_from DESC").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}

func (r *Repository) FindTiersByConfig(configID uuid.UUID) ([]fee.FeeTier, error) {
	var tiers []fee.FeeTier
	err := r.db.Where("fee_config_id = ?", configID).Order("tier_order ASC").Find(&tiers).Error
	return tiers, err
}

func (r *Repository) AcquireAdvisoryLock(tx *gorm.DB, key string) (bool, error) {
	var acquired bool
	err := r.session(tx).Raw("SELECT pg_try_advisory_xact_lock(hashtext(?))", key).Scan(&acquired).Error
	return acquired, err
}

func (r *Repository) ListTransactions(filter ListFilter) ([]Transaction, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	q := r.db.Model(&Transaction{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.ZoneID != nil {
		q = q.Where("zone_id = ?", *filter.ZoneID)
	}
	if filter.StartDate != nil {
		q = q.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		q = q.Where("created_at <= ?", *filter.EndDate)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []Transaction
	err := q.Order("created_at DESC").Offset((filter.Page - 1) * filter.Limit).Limit(filter.Limit).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) UpdateEntryPhotoURL(id uuid.UUID, url string) error {
	return r.db.Model(&Transaction{}).Where("id = ?", id).Updates(map[string]any{
		"entry_photo_url": url,
		"updated_at":      time.Now(),
	}).Error
}

func (r *Repository) UpdateExitPhotoURL(id uuid.UUID, url string) error {
	return r.db.Model(&Transaction{}).Where("id = ?", id).Updates(map[string]any{
		"exit_photo_url": url,
		"updated_at":     time.Now(),
	}).Error
}

func (r *Repository) ListLogs(transactionID uuid.UUID, page, limit int) ([]TransactionLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	q := r.db.Model(&TransactionLog{}).Where("transaction_id = ?", transactionID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []TransactionLog
	err := q.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) session(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}
