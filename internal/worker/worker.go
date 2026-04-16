// internal/worker/worker.go
package worker

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"

	"github.com/yyypluto/parkieee-api/pkg/event"
)

// StartAuditWriter consumes AuditLogEvent from the bus and writes to DB.
// Phase 3 will fill in the DB write logic.
func StartAuditWriter(ctx context.Context, bus *event.Bus) {
	ch := bus.Subscribe()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-ch:
				if ev.Type == "AuditLogEvent" {
					log.Printf("audit worker: received event (stub) %+v", ev.Payload)
				}
			}
		}
	}()
}

// StartNotificationWriter consumes NotificationEvent from the bus.
// Phase 3 will fill in the DB write logic.
func StartNotificationWriter(ctx context.Context, bus *event.Bus) {
	ch := bus.Subscribe()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-ch:
				if ev.Type == "NotificationEvent" {
					log.Printf("notification worker: received event (stub) %+v", ev.Payload)
				}
			}
		}
	}()
}

// HandleAuditEvent processes audit.* outbox events.
func HandleAuditEvent(ctx context.Context, payload json.RawMessage) error {
	slog.InfoContext(ctx, "audit outbox event", "payload", string(payload))
	return nil
}

// HandleNotification handles notify.* outbox events.
func HandleNotification(ctx context.Context, payload json.RawMessage) error {
	slog.InfoContext(ctx, "notification outbox event", "payload", string(payload))
	return nil
}
