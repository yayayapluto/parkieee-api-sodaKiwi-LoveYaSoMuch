package fee

import (
	"errors"
	"sort"
)

var ErrInvalidTierDuration = errors.New("invalid tier duration")

// Calculate computes parking fee from a config and ordered tiers.
// durationMinutes uses total parking duration in minutes.
func Calculate(config FeeConfig, tiers []FeeTier, durationMinutes int) (int, error) {
	if durationMinutes < 0 {
		durationMinutes = 0
	}

	remaining := durationMinutes - config.GracePeriodMinutes
	if remaining < 0 {
		remaining = 0
	}

	total := config.BaseFee
	if len(tiers) == 0 || remaining == 0 {
		return total, nil
	}

	sorted := append([]FeeTier(nil), tiers...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TierOrder < sorted[j].TierOrder
	})

	for _, tier := range sorted {
		if remaining <= 0 {
			break
		}
		if tier.DurationMinutes <= 0 {
			return 0, ErrInvalidTierDuration
		}

		if tier.IsLastTier || remaining <= tier.DurationMinutes {
			units := ceilDiv(remaining, tier.DurationMinutes)
			total += units * tier.FeeAmount
			break
		}

		total += tier.FeeAmount
		remaining -= tier.DurationMinutes
	}

	return total, nil
}

func ceilDiv(value, divisor int) int {
	if divisor <= 0 {
		return 0
	}
	return (value + divisor - 1) / divisor
}
