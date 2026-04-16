package zone

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo Repo
	now  func() time.Time
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo, now: time.Now}
}

type PairRequestResult struct {
	Code      string
	ExpiresAt time.Time
}

type PairVerifyResult struct {
	Status    string
	GateID    *uuid.UUID
	GateToken string
	ExpiresAt time.Time
}

func (s *Service) ListZones() ([]Zone, error) {
	return s.repo.ListZones()
}

func (s *Service) CreateZone(name, description string, capacity, additionalFee int, forVehicleTypeID *uuid.UUID) (*Zone, error) {
	now := s.now()
	zone := &Zone{
		ID:               uuid.New(),
		Name:             strings.TrimSpace(name),
		Description:      strings.TrimSpace(description),
		Capacity:         capacity,
		AdditionalFee:    additionalFee,
		ForVehicleTypeID: forVehicleTypeID,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.CreateZone(zone); err != nil {
		return nil, err
	}
	return zone, nil
}

func (s *Service) GetZone(id uuid.UUID) (*Zone, error) {
	return s.repo.FindZoneByID(id)
}

func (s *Service) UpdateZone(id uuid.UUID, name, description string, capacity, additionalFee int, forVehicleTypeID *uuid.UUID, isActive bool) (*Zone, error) {
	zone, err := s.repo.FindZoneByID(id)
	if err != nil || zone == nil {
		return zone, err
	}

	zone.Name = strings.TrimSpace(name)
	zone.Description = strings.TrimSpace(description)
	zone.Capacity = capacity
	zone.AdditionalFee = additionalFee
	zone.ForVehicleTypeID = forVehicleTypeID
	zone.IsActive = isActive
	zone.UpdatedAt = s.now()

	if err := s.repo.UpdateZone(zone); err != nil {
		return nil, err
	}
	return zone, nil
}

func (s *Service) ListGates() ([]Gate, error) {
	return s.repo.ListGates()
}

func (s *Service) CreateGate(zoneID uuid.UUID, name, gateType, mode, locationDesc string) (*Gate, error) {
	now := s.now()
	gate := &Gate{
		ID:           uuid.New(),
		ZoneID:       zoneID,
		Name:         strings.TrimSpace(name),
		GateType:     strings.TrimSpace(strings.ToLower(gateType)),
		Mode:         strings.TrimSpace(strings.ToLower(mode)),
		LocationDesc: strings.TrimSpace(locationDesc),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreateGate(gate); err != nil {
		return nil, err
	}
	return gate, nil
}

func (s *Service) GetGate(id uuid.UUID) (*Gate, error) {
	return s.repo.FindGateByID(id)
}

func (s *Service) UpdateGate(id, zoneID uuid.UUID, name, gateType, mode, locationDesc string, isActive bool) (*Gate, error) {
	gate, err := s.repo.FindGateByID(id)
	if err != nil || gate == nil {
		return gate, err
	}

	gate.ZoneID = zoneID
	gate.Name = strings.TrimSpace(name)
	gate.GateType = strings.TrimSpace(strings.ToLower(gateType))
	gate.Mode = strings.TrimSpace(strings.ToLower(mode))
	gate.LocationDesc = strings.TrimSpace(locationDesc)
	gate.IsActive = isActive
	gate.UpdatedAt = s.now()

	if err := s.repo.UpdateGate(gate); err != nil {
		return nil, err
	}
	return gate, nil
}

func (s *Service) ListGateDevices(gateID uuid.UUID) ([]GateDevice, error) {
	return s.repo.ListGateDevices(gateID)
}

func (s *Service) UpdateGateDeviceStatus(id uuid.UUID, status, errorMessage string) error {
	return s.repo.UpdateGateDeviceStatus(id, strings.TrimSpace(strings.ToLower(status)), strings.TrimSpace(errorMessage))
}

func (s *Service) RequestPairingCode() (*PairRequestResult, error) {
	now := s.now()
	code := strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:6]
	row := &GatePairingCode{
		ID:        uuid.New(),
		Code:      code,
		Status:    PairingStatusPending,
		ExpiresAt: now.Add(5 * time.Minute),
		CreatedAt: now,
	}
	if err := s.repo.CreatePairingCode(row); err != nil {
		return nil, err
	}
	return &PairRequestResult{Code: row.Code, ExpiresAt: row.ExpiresAt}, nil
}

func (s *Service) ConfirmPairingCode(code string, gateID, confirmedBy uuid.UUID) error {
	return s.repo.ConfirmPairingCode(strings.TrimSpace(strings.ToUpper(code)), gateID, confirmedBy, NewGateToken(), s.now())
}

func (s *Service) VerifyPairingCode(code string) (*PairVerifyResult, error) {
	row, err := s.repo.FindActivePairingCode(strings.TrimSpace(strings.ToUpper(code)), s.now())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &PairVerifyResult{Status: PairingStatusExpired}, nil
	}

	out := &PairVerifyResult{
		Status:    row.Status,
		GateID:    row.GateID,
		GateToken: row.GateJWT,
		ExpiresAt: row.ExpiresAt,
	}
	if s.now().After(row.ExpiresAt) && row.Status == PairingStatusPending {
		out.Status = PairingStatusExpired
	}
	return out, nil
}
