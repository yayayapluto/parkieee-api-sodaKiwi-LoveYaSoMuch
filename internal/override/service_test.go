package override_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/yyypluto/parkieee-api/internal/override"
	overridemocks "github.com/yyypluto/parkieee-api/internal/override/mocks"
	"gorm.io/gorm"
)

func TestApply_DailyLimitExceeded_ReturnsError(t *testing.T) {
	repo := overridemocks.NewRepo(t)
	txID := uuid.New()
	opID := uuid.New()
	maxPerDay := 5

	repo.On("FindTransaction", txID).Return(&override.LocalTx{
		ID:            txID,
		Status:        "pending_payment",
		CalculatedFee: 10000,
	}, nil)
	repo.On("FindActiveConfig").Return(&override.LocalConfig{
		MaxOverridesPerDay: maxPerDay,
	}, nil)
	repo.On("CountTodayByOperator", opID).Return(int64(maxPerDay), nil)

	svc := override.NewService(repo, nil, nil)
	rec, err := svc.Apply(context.Background(), override.ApplyInput{
		TransactionID: txID,
		OperatorID:    opID,
		OverrideType:  "fee_adjust",
		Reason:        "test",
		AdjustedFee:   8000,
	})

	assert.Nil(t, rec)
	assert.ErrorIs(t, err, override.ErrDailyLimitExceeded)
}

func TestApply_FeeAdjust_CreatesOverrideAndUpdatesTransaction(t *testing.T) {
	repo := overridemocks.NewRepo(t)
	txID := uuid.New()
	opID := uuid.New()
	originalFee := 10000
	adjustedFee := 8000

	repo.On("FindTransaction", txID).Return(&override.LocalTx{
		ID:            txID,
		Status:        "pending_payment",
		CalculatedFee: originalFee,
	}, nil)
	repo.On("FindActiveConfig").Return(&override.LocalConfig{
		MaxOverridesPerDay: 10,
	}, nil)
	repo.On("CountTodayByOperator", opID).Return(int64(2), nil)
	repo.On("Create", (*gorm.DB)(nil), mock.MatchedBy(func(o *override.OperatorOverride) bool {
		return o.TransactionID == txID &&
			o.OperatorID == opID &&
			o.OverrideType == "fee_adjust" &&
			o.OriginalFee == originalFee &&
			o.AdjustedFee == adjustedFee
	})).Return(nil)
	repo.On("UpdateTransactionFee", (*gorm.DB)(nil), txID, adjustedFee).Return(nil)

	svc := override.NewService(repo, nil, nil)
	rec, err := svc.Apply(context.Background(), override.ApplyInput{
		TransactionID: txID,
		OperatorID:    opID,
		OverrideType:  "fee_adjust",
		Reason:        "customer complaint",
		AdjustedFee:   adjustedFee,
	})

	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, originalFee, rec.OriginalFee)
	assert.Equal(t, adjustedFee, rec.AdjustedFee)
	assert.Equal(t, "fee_adjust", rec.OverrideType)
}

func TestApply_BarrierRelease_DoesNotUpdateFee(t *testing.T) {
	repo := overridemocks.NewRepo(t)
	txID := uuid.New()
	opID := uuid.New()
	exitGateID := uuid.New()
	originalFee := 10000

	repo.On("FindTransaction", txID).Return(&override.LocalTx{
		ID:            txID,
		Status:        "pending_payment",
		CalculatedFee: originalFee,
		ExitGateID:    &exitGateID,
	}, nil)
	repo.On("FindActiveConfig").Return(&override.LocalConfig{
		MaxOverridesPerDay: 10,
	}, nil)
	repo.On("CountTodayByOperator", opID).Return(int64(0), nil)
	repo.On("Create", (*gorm.DB)(nil), mock.MatchedBy(func(o *override.OperatorOverride) bool {
		return o.TransactionID == txID &&
			o.OverrideType == "barrier_release" &&
			o.AdjustedFee == originalFee // no fee change
	})).Return(nil)
	// UpdateTransactionFee must NOT be called — no On() registration for it

	svc := override.NewService(repo, nil, nil)
	rec, err := svc.Apply(context.Background(), override.ApplyInput{
		TransactionID: txID,
		OperatorID:    opID,
		OverrideType:  "barrier_release",
		Reason:        "gate stuck",
		AdjustedFee:   0, // ignored for barrier_release
	})

	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "barrier_release", rec.OverrideType)
	assert.Equal(t, originalFee, rec.AdjustedFee)
}

func TestApply_TransactionNotFound_ReturnsError(t *testing.T) {
	repo := overridemocks.NewRepo(t)
	txID := uuid.New()
	opID := uuid.New()

	repo.On("FindTransaction", txID).Return((*override.LocalTx)(nil), nil)

	svc := override.NewService(repo, nil, nil)
	rec, err := svc.Apply(context.Background(), override.ApplyInput{
		TransactionID: txID,
		OperatorID:    opID,
		OverrideType:  "fee_adjust",
		Reason:        "test",
	})

	assert.Nil(t, rec)
	assert.ErrorIs(t, err, override.ErrTransactionNotFound)
}

func TestApply_InvalidOverrideType_ReturnsError(t *testing.T) {
	repo := overridemocks.NewRepo(t)
	txID := uuid.New()
	opID := uuid.New()

	svc := override.NewService(repo, nil, nil)
	rec, err := svc.Apply(context.Background(), override.ApplyInput{
		TransactionID: txID,
		OperatorID:    opID,
		OverrideType:  "invalid_type",
		Reason:        "test",
	})

	assert.Nil(t, rec)
	assert.ErrorIs(t, err, override.ErrInvalidOverrideType)
}
