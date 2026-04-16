package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListVehicleTypes(filter VehicleTypeListFilter) ([]VehicleType, int64, error) {
	q := r.db.Model(&VehicleType{})
	if filter.Search != "" {
		like := "%" + strings.ToLower(filter.Search) + "%"
		q = q.Where("LOWER(name) LIKE ?", like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []VehicleType
	err := q.Order("created_at DESC").
		Offset((filter.Page - 1) * filter.Limit).
		Limit(filter.Limit).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) FindVehicleTypeByID(id uuid.UUID) (*VehicleType, error) {
	var vt VehicleType
	err := r.db.Where("id = ?", id).First(&vt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &vt, err
}

func (r *Repository) FindVehicleTypeByName(name string) (*VehicleType, error) {
	var vt VehicleType
	err := r.db.Where("LOWER(name) = ?", strings.ToLower(name)).First(&vt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &vt, err
}

func (r *Repository) CreateVehicleType(vt *VehicleType) error {
	return r.db.Create(vt).Error
}

func (r *Repository) UpdateVehicleType(vt *VehicleType) error {
	return r.db.Save(vt).Error
}

func (r *Repository) DeleteVehicleType(id uuid.UUID) error {
	return r.db.Delete(&VehicleType{}, "id = ?", id).Error
}

func (r *Repository) VehicleTypeInUse(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&Vehicle{}).Where("vehicle_type_id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *Repository) ListVehicles(filter VehicleListFilter) ([]Vehicle, int64, error) {
	q := r.db.Model(&Vehicle{})

	if filter.PlateNumber != "" {
		like := "%" + strings.ToLower(filter.PlateNumber) + "%"
		q = q.Where("LOWER(plate_number) LIKE ?", like)
	}
	if filter.VehicleTypeID != nil {
		q = q.Where("vehicle_type_id = ?", *filter.VehicleTypeID)
	}
	if filter.Search != "" {
		like := "%" + strings.ToLower(filter.Search) + "%"
		q = q.Where("LOWER(plate_number) LIKE ? OR LOWER(notes) LIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []Vehicle
	err := q.Preload("VehicleType").
		Order("created_at DESC").
		Offset((filter.Page - 1) * filter.Limit).
		Limit(filter.Limit).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) FindVehicleByID(id uuid.UUID) (*Vehicle, error) {
	var v Vehicle
	err := r.db.Preload("VehicleType").Where("id = ?", id).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &v, err
}

func (r *Repository) CreateVehicle(v *Vehicle) error {
	return r.db.Create(v).Error
}

func (r *Repository) UpdateVehicle(v *Vehicle) error {
	v.UpdatedAt = time.Now()
	return r.db.Save(v).Error
}

func (r *Repository) ListRFIDCardsByVehicle(vehicleID uuid.UUID, isActive *bool) ([]RFIDCard, error) {
	q := r.db.Model(&RFIDCard{}).Where("vehicle_id = ?", vehicleID)
	if isActive != nil {
		q = q.Where("is_active = ?", *isActive)
	}

	var rows []RFIDCard
	err := q.Order("created_at DESC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repository) FindRFIDByCardUID(cardUID string) (*RFIDCard, error) {
	var c RFIDCard
	err := r.db.Where("card_uid = ?", cardUID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (r *Repository) FindRFIDByID(id uuid.UUID) (*RFIDCard, error) {
	var c RFIDCard
	err := r.db.Where("id = ?", id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (r *Repository) CreateRFIDCard(c *RFIDCard) error {
	return r.db.Create(c).Error
}

func (r *Repository) DeactivateRFID(id, actorID uuid.UUID, deactivatedAt time.Time) error {
	return r.db.Model(&RFIDCard{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"is_active":      false,
			"deactivated_at": deactivatedAt,
			"deactivated_by": actorID,
		}).Error
}
