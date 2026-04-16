package fee_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/internal/fee"
)

type fakeRepo struct {
	systemConfigs map[string]string
	feeConfigs    map[uuid.UUID]fee.FeeConfig
	tiers         map[uuid.UUID]fee.FeeTier
	holidayRates  map[uuid.UUID]fee.HolidayRate
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		systemConfigs: map[string]string{"payment_timeout_minutes": "15"},
		feeConfigs:    map[uuid.UUID]fee.FeeConfig{},
		tiers:         map[uuid.UUID]fee.FeeTier{},
		holidayRates:  map[uuid.UUID]fee.HolidayRate{},
	}
}

func (f *fakeRepo) ListSystemConfigs() ([]fee.SystemConfig, error) {
	out := make([]fee.SystemConfig, 0, len(f.systemConfigs))
	for k, v := range f.systemConfigs {
		out = append(out, fee.SystemConfig{Key: k, Value: v, UpdatedAt: time.Now()})
	}
	return out, nil
}

func (f *fakeRepo) UpdateSystemConfig(key, value string) error {
	f.systemConfigs[key] = value
	return nil
}

func (f *fakeRepo) ListFeeConfigs() ([]fee.FeeConfig, error) {
	out := make([]fee.FeeConfig, 0, len(f.feeConfigs))
	for _, v := range f.feeConfigs {
		out = append(out, v)
	}
	return out, nil
}

func (f *fakeRepo) CreateFeeConfig(cfg *fee.FeeConfig) error {
	f.feeConfigs[cfg.ID] = *cfg
	return nil
}

func (f *fakeRepo) FindFeeConfigByID(id uuid.UUID) (*fee.FeeConfig, error) {
	row, ok := f.feeConfigs[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeRepo) UpdateFeeConfig(cfg *fee.FeeConfig) error {
	f.feeConfigs[cfg.ID] = *cfg
	return nil
}

func (f *fakeRepo) DeleteFeeConfig(id uuid.UUID) error {
	delete(f.feeConfigs, id)
	return nil
}

func (f *fakeRepo) ListFeeTiers(feeConfigID uuid.UUID) ([]fee.FeeTier, error) {
	out := make([]fee.FeeTier, 0)
	for _, v := range f.tiers {
		if v.FeeConfigID == feeConfigID {
			out = append(out, v)
		}
	}
	return out, nil
}

func (f *fakeRepo) CreateFeeTier(tier *fee.FeeTier) error {
	f.tiers[tier.ID] = *tier
	return nil
}

func (f *fakeRepo) FindFeeTierByID(id uuid.UUID) (*fee.FeeTier, error) {
	row, ok := f.tiers[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeRepo) UpdateFeeTier(tier *fee.FeeTier) error {
	f.tiers[tier.ID] = *tier
	return nil
}

func (f *fakeRepo) DeleteFeeTier(id uuid.UUID) error {
	delete(f.tiers, id)
	return nil
}

func (f *fakeRepo) ListHolidayRates() ([]fee.HolidayRate, error) {
	out := make([]fee.HolidayRate, 0, len(f.holidayRates))
	for _, v := range f.holidayRates {
		out = append(out, v)
	}
	return out, nil
}

func (f *fakeRepo) CreateHolidayRate(rate *fee.HolidayRate) error {
	f.holidayRates[rate.ID] = *rate
	return nil
}

func (f *fakeRepo) FindHolidayRateByID(id uuid.UUID) (*fee.HolidayRate, error) {
	row, ok := f.holidayRates[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeRepo) UpdateHolidayRate(rate *fee.HolidayRate) error {
	f.holidayRates[rate.ID] = *rate
	return nil
}

func (f *fakeRepo) DeleteHolidayRate(id uuid.UUID) error {
	delete(f.holidayRates, id)
	return nil
}

func TestListSystemConfigs_ReturnsList(t *testing.T) {
	repo := newFakeRepo()
	svc := fee.NewService(repo)

	rows, err := svc.ListSystemConfigs()
	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "payment_timeout_minutes", rows[0].Key)
}

func TestUpdateSystemConfig_CallsRepo(t *testing.T) {
	repo := newFakeRepo()
	svc := fee.NewService(repo)

	err := svc.UpdateSystemConfig("payment_timeout_minutes", "20")
	assert.NoError(t, err)
	assert.Equal(t, "20", repo.systemConfigs["payment_timeout_minutes"])
}

func TestFeeConfigCrud(t *testing.T) {
	repo := newFakeRepo()
	svc := fee.NewService(repo)
	zoneID := uuid.New()
	vehicleTypeID := uuid.New()
	now := time.Now()

	created, err := svc.CreateFeeConfig(fee.FeeConfigInput{
		ZoneID:             zoneID,
		VehicleTypeID:      vehicleTypeID,
		BaseFee:            5000,
		GracePeriodMinutes: 10,
		IsActive:           true,
		EffectiveFrom:      now,
	})
	assert.NoError(t, err)
	assert.NotNil(t, created)

	got, err := svc.GetFeeConfig(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, 5000, got.BaseFee)

	updated, err := svc.UpdateFeeConfig(created.ID, fee.FeeConfigInput{
		ZoneID:             zoneID,
		VehicleTypeID:      vehicleTypeID,
		BaseFee:            7000,
		GracePeriodMinutes: 5,
		IsActive:           true,
		EffectiveFrom:      now,
	})
	assert.NoError(t, err)
	assert.Equal(t, 7000, updated.BaseFee)

	err = svc.DeleteFeeConfig(created.ID)
	assert.NoError(t, err)
	_, err = svc.GetFeeConfig(created.ID)
	assert.ErrorIs(t, err, fee.ErrFeeConfigNotFound)
}

func TestFeeTierCrud(t *testing.T) {
	repo := newFakeRepo()
	svc := fee.NewService(repo)
	zoneID := uuid.New()
	vehicleTypeID := uuid.New()
	now := time.Now()

	createdCfg, err := svc.CreateFeeConfig(fee.FeeConfigInput{
		ZoneID:             zoneID,
		VehicleTypeID:      vehicleTypeID,
		BaseFee:            5000,
		GracePeriodMinutes: 10,
		IsActive:           true,
		EffectiveFrom:      now,
	})
	assert.NoError(t, err)

	createdTier, err := svc.CreateFeeTier(createdCfg.ID, fee.FeeTierInput{TierOrder: 1, DurationMinutes: 60, FeeAmount: 2000, IsLastTier: false})
	assert.NoError(t, err)
	assert.NotNil(t, createdTier)

	rows, err := svc.ListFeeTiers(createdCfg.ID)
	assert.NoError(t, err)
	assert.Len(t, rows, 1)

	updated, err := svc.UpdateFeeTier(createdCfg.ID, createdTier.ID, fee.FeeTierInput{TierOrder: 2, DurationMinutes: 120, FeeAmount: 3000, IsLastTier: true})
	assert.NoError(t, err)
	assert.Equal(t, 2, updated.TierOrder)

	err = svc.DeleteFeeTier(createdCfg.ID, createdTier.ID)
	assert.NoError(t, err)
}

func TestHolidayRateCrud(t *testing.T) {
	repo := newFakeRepo()
	svc := fee.NewService(repo)

	created, err := svc.CreateHolidayRate(fee.HolidayRateInput{
		Name:          "Lebaran",
		HolidayDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		AdditionalFee: 2000,
		Multiplier:    1.5,
		IsActive:      true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, created)

	got, err := svc.GetHolidayRate(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Lebaran", got.Name)

	updated, err := svc.UpdateHolidayRate(created.ID, fee.HolidayRateInput{
		Name:          "Lebaran Update",
		HolidayDate:   time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
		AdditionalFee: 3000,
		Multiplier:    1.2,
		IsActive:      true,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Lebaran Update", updated.Name)

	err = svc.DeleteHolidayRate(created.ID)
	assert.NoError(t, err)
	_, err = svc.GetHolidayRate(created.ID)
	assert.ErrorIs(t, err, fee.ErrHolidayRateNotFound)
}
