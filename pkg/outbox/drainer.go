package outbox

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Start launches a background drainer that processes pending outbox events.
func Start(ctx context.Context, db *gorm.DB, handlers map[string]HandlerFunc) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				drain(ctx, db, handlers)
			}
		}
	}()

	return done
}

func drain(ctx context.Context, db *gorm.DB, handlers map[string]HandlerFunc) {
	now := time.Now()
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var events []OutboxEvent
		if err := tx.
			Where("processed_at IS NULL").
			Order("created_at ASC").
			Limit(50).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Find(&events).Error; err != nil {
			return err
		}

		for _, ev := range events {
			if processEvent(ctx, ev, handlers) {
				if err := tx.Model(&OutboxEvent{}).Where("id = ?", ev.ID).Update("processed_at", now).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		slog.Error("outbox: drain failed", "error", err)
	}
}
