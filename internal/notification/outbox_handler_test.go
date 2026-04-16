package notification_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yyypluto/parkieee-api/internal/notification"
	notificationmocks "github.com/yyypluto/parkieee-api/internal/notification/mocks"
)

func TestHandleEventFn_WritesNotification(t *testing.T) {
	mockRepo := notificationmocks.NewRepo(t)
	payload := json.RawMessage(`{"gate_id":"abc","plate":"B1234XY"}`)

	mockRepo.On("Create", mock.MatchedBy(func(n *notification.Notification) bool {
		return n.Type == "plate_mismatch" &&
			n.Title == "Plate Mismatch Detected" &&
			n.Body == string(payload)
	})).Return(nil)

	fn := notification.HandleEventFn(mockRepo, "plate_mismatch", "Plate Mismatch Detected")
	err := fn(context.Background(), payload)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleEventFn_RepoError_Propagates(t *testing.T) {
	mockRepo := notificationmocks.NewRepo(t)
	payload := json.RawMessage(`{"gate_id":"abc"}`)

	mockRepo.On("Create", mock.AnythingOfType("*notification.Notification")).
		Return(errors.New("db down"))

	fn := notification.HandleEventFn(mockRepo, "unclosed_flag", "Unclosed Transaction Flagged")
	err := fn(context.Background(), payload)

	assert.EqualError(t, err, "db down")
}
