package fee

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculate_WithinGracePeriodReturnsBaseFee(t *testing.T) {
	config := FeeConfig{BaseFee: 2000, GracePeriodMinutes: 10}

	amount, err := Calculate(config, nil, 8)
	require.NoError(t, err)
	assert.Equal(t, 2000, amount)
}

func TestCalculate_MultipleTiersAccumulatesFee(t *testing.T) {
	config := FeeConfig{BaseFee: 2000, GracePeriodMinutes: 10}
	tiers := []FeeTier{
		{TierOrder: 2, DurationMinutes: 60, FeeAmount: 2000, IsLastTier: true},
		{TierOrder: 1, DurationMinutes: 60, FeeAmount: 3000, IsLastTier: false},
	}

	amount, err := Calculate(config, tiers, 130)
	require.NoError(t, err)
	assert.Equal(t, 7000, amount)
}

func TestCalculate_LastTierChargesProRata(t *testing.T) {
	config := FeeConfig{BaseFee: 2000, GracePeriodMinutes: 10}
	tiers := []FeeTier{
		{TierOrder: 1, DurationMinutes: 60, FeeAmount: 3000, IsLastTier: false},
		{TierOrder: 2, DurationMinutes: 60, FeeAmount: 2000, IsLastTier: true},
	}

	amount, err := Calculate(config, tiers, 170)
	require.NoError(t, err)
	assert.Equal(t, 9000, amount)
}

func TestCalculate_InvalidTierDuration(t *testing.T) {
	config := FeeConfig{BaseFee: 2000, GracePeriodMinutes: 0}
	tiers := []FeeTier{
		{TierOrder: 1, DurationMinutes: 0, FeeAmount: 1000, IsLastTier: true},
	}

	amount, err := Calculate(config, tiers, 30)
	assert.ErrorIs(t, err, ErrInvalidTierDuration)
	assert.Equal(t, 0, amount)
}
