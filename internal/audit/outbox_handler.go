package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
)

// HandleEventFn returns an outbox.HandlerFunc that writes one AuditLog row.
func HandleEventFn(repo Repo, action, entityType string) outbox.HandlerFunc {
	return func(ctx context.Context, payload json.RawMessage) error {
		var extracted struct {
			ID uuid.UUID `json:"id"`
		}
		if err := json.Unmarshal(payload, &extracted); err != nil {
			return fmt.Errorf("audit: malformed payload: %w", err)
		}

		var entityID *uuid.UUID
		if extracted.ID != uuid.Nil {
			entityID = &extracted.ID
		}

		return repo.Create(&AuditLog{
			ID:         uuid.New(),
			Action:     action,
			EntityType: entityType,
			EntityID:   entityID,
			Metadata:   payload,
			CreatedAt:  time.Now(),
			// TODO: extract UserID from ctx once request context carries user identity
			// TODO: extract IPAddress from ctx once request context carries IP
		})
	}
}
