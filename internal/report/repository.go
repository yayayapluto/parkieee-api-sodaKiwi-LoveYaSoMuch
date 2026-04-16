package report

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DailyRevenue maps to the mv_daily_revenue materialized view.
type DailyRevenue struct {
	Day              time.Time `gorm:"column:day"              json:"day"`
	ZoneID           uuid.UUID `gorm:"column:zone_id"          json:"zone_id"`
	VehicleTypeID    uuid.UUID `gorm:"column:vehicle_type_id"  json:"vehicle_type_id"`
	TransactionCount int       `gorm:"column:transaction_count" json:"transaction_count"`
	TotalRevenue     int       `gorm:"column:total_revenue"    json:"total_revenue"`
}

func (DailyRevenue) TableName() string { return "mv_daily_revenue" }

// ZoneOccupancy maps to the mv_zone_occupancy materialized view.
type ZoneOccupancy struct {
	ZoneID   uuid.UUID `gorm:"column:zone_id"   json:"zone_id"`
	ZoneName string    `gorm:"column:zone_name" json:"zone_name"`
	Capacity int       `gorm:"column:capacity"  json:"capacity"`
	Occupied int       `gorm:"column:occupied"  json:"occupied"`
}

func (ZoneOccupancy) TableName() string { return "mv_zone_occupancy" }

// RevenueFilter holds optional query constraints.
type RevenueFilter struct {
	From          time.Time
	To            time.Time
	ZoneID        *uuid.UUID
	VehicleTypeID *uuid.UUID
}

type Repo interface {
	DailyRevenue(from, to time.Time, zoneID, vehicleTypeID *uuid.UUID) ([]DailyRevenue, error)
	ZoneOccupancy() ([]ZoneOccupancy, error)
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) DailyRevenue(from, to time.Time, zoneID, vehicleTypeID *uuid.UUID) ([]DailyRevenue, error) {
	q := r.db.Model(&DailyRevenue{}).Order("day DESC")
	if !from.IsZero() {
		q = q.Where("day >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("day <= ?", to)
	}
	if zoneID != nil {
		q = q.Where("zone_id = ?", zoneID)
	}
	if vehicleTypeID != nil {
		q = q.Where("vehicle_type_id = ?", vehicleTypeID)
	}
	var rows []DailyRevenue
	return rows, q.Find(&rows).Error
}

func (r *Repository) ZoneOccupancy() ([]ZoneOccupancy, error) {
	var rows []ZoneOccupancy
	return rows, r.db.Find(&rows).Error
}
