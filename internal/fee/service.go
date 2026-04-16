package fee

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct{ repo Repo }

func NewService(repo Repo) *Service { return &Service{repo: repo} }

var (
	ErrValidation         = errors.New("validation error")
	ErrFeeConfigNotFound  = errors.New("fee config not found")
	ErrFeeTierNotFound    = errors.New("fee tier not found")
	ErrHolidayRateNotFound = errors.New("holiday rate not found")
)

type FeeConfigInput struct {
	ZoneID             uuid.UUID
	VehicleTypeID      uuid.UUID
	BaseFee            int
	GracePeriodMinutes int
	IsActive           bool
	EffectiveFrom      time.Time
	EffectiveUntil     *time.Time
}

type FeeTierInput struct {
	TierOrder       int
	DurationMinutes int
	FeeAmount       int
	IsLastTier      bool
}

type HolidayRateInput struct {
	Name          string
	HolidayDate   time.Time
	AdditionalFee int
	Multiplier    float64
	IsActive      bool
}

func (s *Service) ListFeeConfigs() ([]FeeConfig, error) {
	return s.repo.ListFeeConfigs()
}

func (s *Service) CreateFeeConfig(in FeeConfigInput) (*FeeConfig, error) {
	if in.ZoneID == uuid.Nil || in.VehicleTypeID == uuid.Nil || in.BaseFee < 0 || in.GracePeriodMinutes < 0 || in.EffectiveFrom.IsZero() {
		return nil, ErrValidation
	}
	now := time.Now()
	row := &FeeConfig{
		ID:                 uuid.New(),
		ZoneID:             in.ZoneID,
		VehicleTypeID:      in.VehicleTypeID,
		BaseFee:            in.BaseFee,
		GracePeriodMinutes: in.GracePeriodMinutes,
		IsActive:           in.IsActive,
		EffectiveFrom:      in.EffectiveFrom,
		EffectiveUntil:     in.EffectiveUntil,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.CreateFeeConfig(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) GetFeeConfig(id uuid.UUID) (*FeeConfig, error) {
	if id == uuid.Nil {
		return nil, ErrValidation
	}
	row, err := s.repo.FindFeeConfigByID(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrFeeConfigNotFound
	}
	return row, nil
}

func (s *Service) UpdateFeeConfig(id uuid.UUID, in FeeConfigInput) (*FeeConfig, error) {
	if id == uuid.Nil || in.ZoneID == uuid.Nil || in.VehicleTypeID == uuid.Nil || in.BaseFee < 0 || in.GracePeriodMinutes < 0 || in.EffectiveFrom.IsZero() {
		return nil, ErrValidation
	}
	row, err := s.repo.FindFeeConfigByID(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrFeeConfigNotFound
	}
	row.ZoneID = in.ZoneID
	row.VehicleTypeID = in.VehicleTypeID
	row.BaseFee = in.BaseFee
	row.GracePeriodMinutes = in.GracePeriodMinutes
	row.IsActive = in.IsActive
	row.EffectiveFrom = in.EffectiveFrom
	row.EffectiveUntil = in.EffectiveUntil
	row.UpdatedAt = time.Now()
	if err := s.repo.UpdateFeeConfig(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) DeleteFeeConfig(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrValidation
	}
	row, err := s.repo.FindFeeConfigByID(id)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrFeeConfigNotFound
	}
	return s.repo.DeleteFeeConfig(id)
}

func (s *Service) ListFeeTiers(feeConfigID uuid.UUID) ([]FeeTier, error) {
	if feeConfigID == uuid.Nil {
		return nil, ErrValidation
	}
	cfg, err := s.repo.FindFeeConfigByID(feeConfigID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, ErrFeeConfigNotFound
	}
	return s.repo.ListFeeTiers(feeConfigID)
}

func (s *Service) CreateFeeTier(feeConfigID uuid.UUID, in FeeTierInput) (*FeeTier, error) {
	if feeConfigID == uuid.Nil || in.TierOrder <= 0 || in.DurationMinutes <= 0 || in.FeeAmount < 0 {
		return nil, ErrValidation
	}
	cfg, err := s.repo.FindFeeConfigByID(feeConfigID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, ErrFeeConfigNotFound
	}
	now := time.Now()
	row := &FeeTier{
		ID:              uuid.New(),
		FeeConfigID:     feeConfigID,
		TierOrder:       in.TierOrder,
		DurationMinutes: in.DurationMinutes,
		FeeAmount:       in.FeeAmount,
		IsLastTier:      in.IsLastTier,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.CreateFeeTier(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) UpdateFeeTier(feeConfigID, tierID uuid.UUID, in FeeTierInput) (*FeeTier, error) {
	if feeConfigID == uuid.Nil || tierID == uuid.Nil || in.TierOrder <= 0 || in.DurationMinutes <= 0 || in.FeeAmount < 0 {
		return nil, ErrValidation
	}
	cfg, err := s.repo.FindFeeConfigByID(feeConfigID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, ErrFeeConfigNotFound
	}
	row, err := s.repo.FindFeeTierByID(tierID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrFeeTierNotFound
	}
	if row.FeeConfigID != feeConfigID {
		return nil, ErrValidation
	}
	row.TierOrder = in.TierOrder
	row.DurationMinutes = in.DurationMinutes
	row.FeeAmount = in.FeeAmount
	row.IsLastTier = in.IsLastTier
	row.UpdatedAt = time.Now()
	if err := s.repo.UpdateFeeTier(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) DeleteFeeTier(feeConfigID, tierID uuid.UUID) error {
	if feeConfigID == uuid.Nil || tierID == uuid.Nil {
		return ErrValidation
	}
	cfg, err := s.repo.FindFeeConfigByID(feeConfigID)
	if err != nil {
		return err
	}
	if cfg == nil {
		return ErrFeeConfigNotFound
	}
	row, err := s.repo.FindFeeTierByID(tierID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrFeeTierNotFound
	}
	if row.FeeConfigID != feeConfigID {
		return ErrValidation
	}
	return s.repo.DeleteFeeTier(tierID)
}

func (s *Service) ListHolidayRates() ([]HolidayRate, error) {
	return s.repo.ListHolidayRates()
}

func (s *Service) CreateHolidayRate(in HolidayRateInput) (*HolidayRate, error) {
	if strings.TrimSpace(in.Name) == "" || in.HolidayDate.IsZero() || in.AdditionalFee < 0 || in.Multiplier <= 0 {
		return nil, ErrValidation
	}
	now := time.Now()
	row := &HolidayRate{
		ID:            uuid.New(),
		Name:          strings.TrimSpace(in.Name),
		HolidayDate:   in.HolidayDate,
		AdditionalFee: in.AdditionalFee,
		Multiplier:    in.Multiplier,
		IsActive:      in.IsActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.CreateHolidayRate(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) GetHolidayRate(id uuid.UUID) (*HolidayRate, error) {
	if id == uuid.Nil {
		return nil, ErrValidation
	}
	row, err := s.repo.FindHolidayRateByID(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrHolidayRateNotFound
	}
	return row, nil
}

func (s *Service) UpdateHolidayRate(id uuid.UUID, in HolidayRateInput) (*HolidayRate, error) {
	if id == uuid.Nil || strings.TrimSpace(in.Name) == "" || in.HolidayDate.IsZero() || in.AdditionalFee < 0 || in.Multiplier <= 0 {
		return nil, ErrValidation
	}
	row, err := s.repo.FindHolidayRateByID(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrHolidayRateNotFound
	}
	row.Name = strings.TrimSpace(in.Name)
	row.HolidayDate = in.HolidayDate
	row.AdditionalFee = in.AdditionalFee
	row.Multiplier = in.Multiplier
	row.IsActive = in.IsActive
	row.UpdatedAt = time.Now()
	if err := s.repo.UpdateHolidayRate(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) DeleteHolidayRate(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrValidation
	}
	row, err := s.repo.FindHolidayRateByID(id)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrHolidayRateNotFound
	}
	return s.repo.DeleteHolidayRate(id)
}

func (s *Service) ListSystemConfigs() ([]SystemConfig, error) {
	return s.repo.ListSystemConfigs()
}

func (s *Service) UpdateSystemConfig(key, value string) error {
	return s.repo.UpdateSystemConfig(key, value)
}
