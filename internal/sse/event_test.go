package sse

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventMessage_ReturnsTypedJSON(t *testing.T) {
	msg := NewEventMessage("gate.barrier.open", map[string]any{"gate_id": "g1"})

	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(msg), &out))
	assert.Equal(t, "gate.barrier.open", out["type"])
	assert.NotEmpty(t, out["timestamp"])
	payload, ok := out["payload"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "g1", payload["gate_id"])
}

func TestNormalizeEventMessage_LegacyBarrier(t *testing.T) {
	msg := NormalizeEventMessage("open_barrier")

	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(msg), &out))
	assert.Equal(t, "gate.barrier.open", out["type"])
}

func TestNormalizeEventMessage_AlreadyTypedPassThrough(t *testing.T) {
	in := `{"type":"cashier.plate_mismatch","timestamp":"2026-04-16T10:00:00Z","payload":{"transaction_id":"x"}}`
	out := NormalizeEventMessage(in)
	assert.JSONEq(t, in, out)
}
