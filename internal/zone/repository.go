package zone

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo interface {
	ListZones() ([]Zone, error)
	CreateZone(z *Zone) error
	FindZoneByID(id uuid.UUID) (*Zone, error)
	UpdateZone(z *Zone) error

	ListGates() ([]Gate, error)
	CreateGate(g *Gate) error
	FindGateByID(id uuid.UUID) (*Gate, error)
	UpdateGate(g *Gate) error

	ListGateDevices(gateID uuid.UUID) ([]GateDevice, error)
	UpdateGateDeviceStatus(id uuid.UUID, status, errorMessage string) error

	CreatePairingCode(pc *GatePairingCode) error
	FindActivePairingCode(code string, now time.Time) (*GatePairingCode, error)
	ConfirmPairingCode(code string, gateID, confirmedBy uuid.UUID, gateToken string, now time.Time) error
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListZones() ([]Zone, error) {
	var zones []Zone
	err := r.db.Order("created_at DESC").Find(&zones).Error
	return zones, err
}

func (r *Repository) CreateZone(z *Zone) error {
	return r.db.Create(z).Error
}

func (r *Repository) FindZoneByID(id uuid.UUID) (*Zone, error) {
	var zone Zone
	err := r.db.Where("id = ?", id).First(&zone).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &zone, err
}

func (r *Repository) UpdateZone(z *Zone) error {
	return r.db.Save(z).Error
}

func (r *Repository) ListGates() ([]Gate, error) {
	var gates []Gate
	err := r.db.Order("created_at DESC").Find(&gates).Error
	return gates, err
}

func (r *Repository) CreateGate(g *Gate) error {
	return r.db.Create(g).Error
}

func (r *Repository) FindGateByID(id uuid.UUID) (*Gate, error) {
	var gate Gate
	err := r.db.Where("id = ?", id).First(&gate).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &gate, err
}

func (r *Repository) UpdateGate(g *Gate) error {
	return r.db.Save(g).Error
}

func (r *Repository) ListGateDevices(gateID uuid.UUID) ([]GateDevice, error) {
	var devices []GateDevice
	err := r.db.Where("gate_id = ?", gateID).Order("updated_at DESC").Find(&devices).Error
	return devices, err
}

func (r *Repository) UpdateGateDeviceStatus(id uuid.UUID, status, errorMessage string) error {
	return r.db.Model(&GateDevice{}).Where("id = ?", id).Updates(map[string]any{
		"status":        status,
		"error_message": errorMessage,
		"updated_at":    time.Now(),
	}).Error
}

func (r *Repository) CreatePairingCode(pc *GatePairingCode) error {
	return r.db.Create(pc).Error
}

func (r *Repository) FindActivePairingCode(code string, now time.Time) (*GatePairingCode, error) {
	var row GatePairingCode
	err := r.db.Where("code = ? AND expires_at > ?", code, now).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *Repository) ConfirmPairingCode(code string, gateID, confirmedBy uuid.UUID, gateToken string, now time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&GatePairingCode{}).
			Where("code = ? AND status = ?", code, PairingStatusPending).
			Updates(map[string]any{
				"status":       PairingStatusConfirmed,
				"gate_id":      gateID,
				"confirmed_by": confirmedBy,
				"confirmed_at": now,
				"gate_jwt":     gateToken,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&Gate{}).Where("id = ?", gateID).Updates(map[string]any{
			"gate_token": gateToken,
			"updated_at": now,
		}).Error
	})
}
