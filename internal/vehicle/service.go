package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation error")
var ErrNotFound = errors.New("not found")
var ErrInUse = errors.New("in use")
var ErrDuplicate = errors.New("duplicate")
var ErrCardAlreadyAssigned = errors.New("card already assigned")

type Repo interface {
	ListVehicleTypes(filter VehicleTypeListFilter) ([]VehicleType, int64, error)
	FindVehicleTypeByID(id uuid.UUID) (*VehicleType, error)
	FindVehicleTypeByName(name string) (*VehicleType, error)
	CreateVehicleType(vt *VehicleType) error
	UpdateVehicleType(vt *VehicleType) error
	DeleteVehicleType(id uuid.UUID) error
	VehicleTypeInUse(id uuid.UUID) (bool, error)

	ListVehicles(filter VehicleListFilter) ([]Vehicle, int64, error)
	FindVehicleByID(id uuid.UUID) (*Vehicle, error)
	CreateVehicle(v *Vehicle) error
	UpdateVehicle(v *Vehicle) error

	ListRFIDCardsByVehicle(vehicleID uuid.UUID, isActive *bool) ([]RFIDCard, error)
	FindRFIDByCardUID(cardUID string) (*RFIDCard, error)
	FindRFIDByID(id uuid.UUID) (*RFIDCard, error)
	CreateRFIDCard(c *RFIDCard) error
	DeactivateRFID(id, actorID uuid.UUID, deactivatedAt time.Time) error
}

type VehicleTypeListFilter struct {
	Page   int
	Limit  int
	Search string
}

type VehicleListFilter struct {
	Page          int
	Limit         int
	PlateNumber   string
	VehicleTypeID *uuid.UUID
	Search        string
}

type VehicleTypeInput struct {
	Name        string
	MinimumFee  int
	Description string
}

type VehicleInput struct {
	PlateNumber   string
	VehicleTypeID uuid.UUID
	Source        string
	Notes         string
}

type VehicleUpdateInput struct {
	VehicleTypeID *uuid.UUID
	Notes         *string
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListVehicleTypes(filter VehicleTypeListFilter) ([]VehicleType, int64, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	return s.repo.ListVehicleTypes(filter)
}

func (s *Service) CreateVehicleType(in VehicleTypeInput) (*VehicleType, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || in.MinimumFee < 0 {
		return nil, ErrValidation
	}

	existing, err := s.repo.FindVehicleTypeByName(name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicate
	}

	vt := &VehicleType{
		ID:          uuid.New(),
		Name:        name,
		MinimumFee:  in.MinimumFee,
		Description: strings.TrimSpace(in.Description),
		CreatedAt:   time.Now(),
	}
	if err := s.repo.CreateVehicleType(vt); err != nil {
		return nil, err
	}
	return vt, nil
}

func (s *Service) UpdateVehicleType(id uuid.UUID, in VehicleTypeInput) (*VehicleType, error) {
	if id == uuid.Nil {
		return nil, ErrValidation
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || in.MinimumFee < 0 {
		return nil, ErrValidation
	}

	vt, err := s.repo.FindVehicleTypeByID(id)
	if err != nil {
		return nil, err
	}
	if vt == nil {
		return nil, ErrNotFound
	}

	duplicate, err := s.repo.FindVehicleTypeByName(name)
	if err != nil {
		return nil, err
	}
	if duplicate != nil && duplicate.ID != id {
		return nil, ErrDuplicate
	}

	vt.Name = name
	vt.MinimumFee = in.MinimumFee
	vt.Description = strings.TrimSpace(in.Description)
	if err := s.repo.UpdateVehicleType(vt); err != nil {
		return nil, err
	}
	return vt, nil
}

func (s *Service) DeleteVehicleType(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrValidation
	}

	vt, err := s.repo.FindVehicleTypeByID(id)
	if err != nil {
		return err
	}
	if vt == nil {
		return ErrNotFound
	}

	inUse, err := s.repo.VehicleTypeInUse(id)
	if err != nil {
		return err
	}
	if inUse {
		return ErrInUse
	}

	return s.repo.DeleteVehicleType(id)
}

func (s *Service) ListVehicles(filter VehicleListFilter) ([]Vehicle, int64, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	return s.repo.ListVehicles(filter)
}

func (s *Service) CreateVehicle(in VehicleInput) (*Vehicle, error) {
	plate := strings.ToUpper(strings.TrimSpace(in.PlateNumber))
	if plate == "" || in.VehicleTypeID == uuid.Nil {
		return nil, ErrValidation
	}

	vehicleType, err := s.repo.FindVehicleTypeByID(in.VehicleTypeID)
	if err != nil {
		return nil, err
	}
	if vehicleType == nil {
		return nil, ErrNotFound
	}

	now := time.Now()
	v := &Vehicle{
		ID:            uuid.New(),
		PlateNumber:   plate,
		VehicleTypeID: &in.VehicleTypeID,
		Source:        strings.TrimSpace(in.Source),
		Notes:         strings.TrimSpace(in.Notes),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.CreateVehicle(v); err != nil {
		return nil, err
	}

	created, err := s.repo.FindVehicleByID(v.ID)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return v, nil
	}
	return created, nil
}

func (s *Service) GetVehicleByID(id uuid.UUID) (*Vehicle, error) {
	if id == uuid.Nil {
		return nil, ErrValidation
	}
	v, err := s.repo.FindVehicleByID(id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}

func (s *Service) UpdateVehicle(id uuid.UUID, in VehicleUpdateInput) (*Vehicle, error) {
	if id == uuid.Nil {
		return nil, ErrValidation
	}

	v, err := s.repo.FindVehicleByID(id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrNotFound
	}

	if in.VehicleTypeID != nil {
		if *in.VehicleTypeID == uuid.Nil {
			return nil, ErrValidation
		}
		vehicleType, err := s.repo.FindVehicleTypeByID(*in.VehicleTypeID)
		if err != nil {
			return nil, err
		}
		if vehicleType == nil {
			return nil, ErrNotFound
		}
		v.VehicleTypeID = in.VehicleTypeID
	}

	if in.Notes != nil {
		v.Notes = strings.TrimSpace(*in.Notes)
	}

	if err := s.repo.UpdateVehicle(v); err != nil {
		return nil, err
	}

	updated, err := s.repo.FindVehicleByID(id)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrNotFound
	}
	return updated, nil
}

func (s *Service) ListRFIDCards(vehicleID uuid.UUID, isActive *bool) ([]RFIDCard, error) {
	if vehicleID == uuid.Nil {
		return nil, ErrValidation
	}

	v, err := s.repo.FindVehicleByID(vehicleID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrNotFound
	}

	return s.repo.ListRFIDCardsByVehicle(vehicleID, isActive)
}

func (s *Service) AssignRFID(vehicleID uuid.UUID, cardUID string) (*RFIDCard, error) {
	if vehicleID == uuid.Nil {
		return nil, ErrValidation
	}
	uid := strings.TrimSpace(cardUID)
	if uid == "" {
		return nil, ErrValidation
	}

	v, err := s.repo.FindVehicleByID(vehicleID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrNotFound
	}

	existing, err := s.repo.FindRFIDByCardUID(uid)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrCardAlreadyAssigned
	}

	card := &RFIDCard{
		ID:        uuid.New(),
		CardUID:   uid,
		VehicleID: vehicleID,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateRFIDCard(card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *Service) DeactivateRFID(cardID, actorID uuid.UUID) error {
	if cardID == uuid.Nil || actorID == uuid.Nil {
		return ErrValidation
	}

	card, err := s.repo.FindRFIDByID(cardID)
	if err != nil {
		return err
	}
	if card == nil {
		return ErrNotFound
	}
	if !card.IsActive {
		return nil
	}

	return s.repo.DeactivateRFID(cardID, actorID, time.Now())
}

func normalizePage(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
