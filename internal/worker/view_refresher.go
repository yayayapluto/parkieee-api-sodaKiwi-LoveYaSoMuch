package worker

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// StartViewRefresher refreshes mv_zone_occupancy every 5 minutes concurrently.
func StartViewRefresher(ctx context.Context, db *gorm.DB) <-chan struct{} {
	done := make(chan struct{})
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer close(done)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := db.WithContext(ctx).Exec(
					"REFRESH MATERIALIZED VIEW CONCURRENTLY mv_zone_occupancy",
				).Error; err != nil {
					slog.ErrorContext(ctx, "view refresher failed", "view", "mv_zone_occupancy", "error", err)
				}
			}
		}
	}()

	return done
}
