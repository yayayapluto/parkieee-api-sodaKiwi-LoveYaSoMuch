package notification

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
)

func HandleEventFn(repo Repo, notifType, title string) outbox.HandlerFunc {
	return func(ctx context.Context, payload json.RawMessage) error {
		return repo.Create(&Notification{
			ID:        uuid.New(),
			Type:      notifType,
			Title:     title,
			Body:      string(payload),
			IsRead:    false,
			CreatedAt: time.Now(),
		})
	}
}
