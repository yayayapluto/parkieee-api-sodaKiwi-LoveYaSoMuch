package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
	"gorm.io/gorm"
)

// unclosedFlag mirrors transaction.UnclosedTransactionFlag to avoid circular import.
type unclosedFlag struct {
	ID             uuid.UUID  `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID  uuid.UUID  `gorm:"column:transaction_id;type:uuid;not null"`
	FlaggedAt      time.Time  `gorm:"column:flagged_at"`
	FlagType       string     `gorm:"column:flag_type;type:varchar(30)"`
	FlagReason     string     `gorm:"column:flag_reason;type:varchar(100)"`
	Resolved       bool       `gorm:"column:resolved;default:false"`
	ResolvedAt     *time.Time `gorm:"column:resolved_at"`
	ResolvedBy     *uuid.UUID `gorm:"column:resolved_by;type:uuid"`
	ResolutionNote string     `gorm:"column:resolution_note;type:text"`
}

func (unclosedFlag) TableName() string { return "unclosed_transaction_flags" }

// StartUnclosedChecker runs a ticker every intervalMinutes and flags stale active transactions.
// A transaction is stale when: status='active' AND entry_at < now()-4h AND no unresolved flag exists.
func StartUnclosedChecker(ctx context.Context, db *gorm.DB, intervalMinutes int) <-chan struct{} {
	if intervalMinutes <= 0 {
		intervalMinutes = 60
	}

	done := make(chan struct{})
	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	go func() {
		defer close(done)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checkUnclosed(ctx, db)
			}
		}
	}()

	return done
}

func checkUnclosed(ctx context.Context, db *gorm.DB) {
	threshold := time.Now().Add(-4 * time.Hour)
	slog.InfoContext(ctx, "unclosed checker running", "threshold", threshold)

	var ids []uuid.UUID
	err := db.WithContext(ctx).
		Table("transactions").
		Select("id").
		Where("status = ? AND entry_at < ?", "active", threshold).
		Where("id NOT IN (?)",
			db.Table("unclosed_transaction_flags").
				Select("transaction_id").
				Where("resolved = false"),
		).
		Scan(&ids).Error
	if err != nil {
		slog.ErrorContext(ctx, "unclosed checker query failed", "error", err)
		return
	}

	now := time.Now()
	for _, txID := range ids {
		id := txID // capture loop var
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			flag := unclosedFlag{
				ID:            uuid.New(),
				TransactionID: id,
				FlaggedAt:     now,
				FlagType:      "stale",
				FlagReason:    "transaction active for over 4 hours",
				Resolved:      false,
			}
			if err := tx.Create(&flag).Error; err != nil {
				return err
			}
			return outbox.Publish(tx, "notify.unclosed_flag", map[string]any{
				"transaction_id": id,
				"flagged_at":     now,
			})
		})
		if err != nil {
			slog.ErrorContext(ctx, "unclosed checker flag failed", "tx_id", id, "error", err)
		} else {
			slog.InfoContext(ctx, "unclosed transaction flagged", "tx_id", id)
		}
	}
}
