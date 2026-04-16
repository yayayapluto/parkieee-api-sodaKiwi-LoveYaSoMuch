package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OutboxEvent stores transactional side effects that are processed asynchronously.
type OutboxEvent struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventType   string     `gorm:"type:varchar(50);not null"`
	Payload     []byte     `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time  `gorm:"not null;default:now()"`
	ProcessedAt *time.Time `gorm:"index"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}

func AllModels() []any {
	return []any{&OutboxEvent{}}
}

// HandlerFunc handles one outbox payload by event type.
type HandlerFunc func(ctx context.Context, payload json.RawMessage) error

// Publish writes one outbox row in the same DB transaction as the caller.
func Publish(tx *gorm.DB, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return tx.Create(&OutboxEvent{
		EventType: eventType,
		Payload:   data,
	}).Error
}

// processEvent dispatches one event and returns whether it should be marked processed.
func processEvent(ctx context.Context, ev OutboxEvent, handlers map[string]HandlerFunc) bool {
	handler, ok := handlers[ev.EventType]
	if !ok {
		slog.Warn("outbox: no handler registered, marking processed", "event_type", ev.EventType)
		return true
	}

	if err := handler(ctx, json.RawMessage(ev.Payload)); err != nil {
		slog.Error("outbox: handler failed", "event_type", ev.EventType, "event_id", ev.ID, "error", err)
		return false
	}

	return true
}
