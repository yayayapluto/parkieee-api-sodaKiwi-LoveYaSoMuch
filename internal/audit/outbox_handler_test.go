package audit_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yyypluto/parkieee-api/internal/audit"
	auditmocks "github.com/yyypluto/parkieee-api/internal/audit/mocks"
)

func TestHandleEventFn_WritesCorrectFields(t *testing.T) {
	id := uuid.New()
	payload, _ := json.Marshal(map[string]any{"id": id.String()})
	mockRepo := auditmocks.NewRepo(t)
	mockRepo.On("Create", mock.MatchedBy(func(l *audit.AuditLog) bool {
		return l.EntityID != nil && *l.EntityID == id &&
			l.Action == "transaction.entry" &&
			l.EntityType == "transaction"
	})).Return(nil)

	fn := audit.HandleEventFn(mockRepo, "transaction.entry", "transaction")
	err := fn(context.Background(), json.RawMessage(payload))
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleEventFn_NoIDInPayload_StillWrites(t *testing.T) {
	payload := json.RawMessage(`{"other":"field"}`)
	mockRepo := auditmocks.NewRepo(t)
	mockRepo.On("Create", mock.MatchedBy(func(l *audit.AuditLog) bool {
		return l.EntityID == nil && l.Action == "payment.completed"
	})).Return(nil)

	fn := audit.HandleEventFn(mockRepo, "payment.completed", "payment")
	err := fn(context.Background(), payload)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleEventFn_RepoError_PropagatesError(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{"id": uuid.New().String()})
	mockRepo := auditmocks.NewRepo(t)
	mockRepo.On("Create", mock.AnythingOfType("*audit.AuditLog")).
		Return(errors.New("db down"))

	fn := audit.HandleEventFn(mockRepo, "transaction.entry", "transaction")
	err := fn(context.Background(), json.RawMessage(payload))
	assert.EqualError(t, err, "db down")
}
