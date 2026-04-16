package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProcessEvent_NoHandlerMarkedProcessed(t *testing.T) {
	ev := OutboxEvent{ID: uuid.New(), EventType: "unknown", Payload: []byte(`{}`)}
	processed := processEvent(context.Background(), ev, map[string]HandlerFunc{})
	assert.True(t, processed)
}

func TestProcessEvent_HandlerErrorNotProcessed(t *testing.T) {
	ev := OutboxEvent{ID: uuid.New(), EventType: "audit.test", Payload: []byte(`{}`)}
	handlers := map[string]HandlerFunc{
		"audit.test": func(context.Context, json.RawMessage) error {
			return errors.New("boom")
		},
	}

	processed := processEvent(context.Background(), ev, handlers)
	assert.False(t, processed)
}

func TestProcessEvent_HandlerSuccessProcessed(t *testing.T) {
	ev := OutboxEvent{ID: uuid.New(), EventType: "audit.test", Payload: []byte(`{"id":"1"}`)}
	called := false
	handlers := map[string]HandlerFunc{
		"audit.test": func(_ context.Context, payload json.RawMessage) error {
			called = true
			assert.JSONEq(t, `{"id":"1"}`, string(payload))
			return nil
		},
	}

	processed := processEvent(context.Background(), ev, handlers)
	assert.True(t, processed)
	assert.True(t, called)
}
