package sse

import (
	"encoding/json"
	"strings"
	"time"
)

type Event struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Payload   any    `json:"payload"`
}

func NewEventMessage(eventType string, payload any) string {
	msg, err := json.Marshal(Event{
		Type:      strings.TrimSpace(eventType),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	})
	if err != nil {
		return `{"type":"internal.marshal_error","timestamp":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","payload":{}}`
	}
	return string(msg)
}

func NormalizeEventMessage(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return NewEventMessage("legacy.empty", map[string]any{})
	}

	var parsed map[string]any
	if json.Unmarshal([]byte(raw), &parsed) == nil {
		if _, hasType := parsed["type"]; hasType {
			if _, hasTimestamp := parsed["timestamp"]; hasTimestamp {
				if _, hasPayload := parsed["payload"]; hasPayload {
					return raw
				}
			}
		}
	}

	if raw == "open_barrier" {
		return NewEventMessage("gate.barrier.open", map[string]any{"command": "open_barrier"})
	}
	if strings.HasPrefix(raw, "plate_mismatch:") {
		parts := strings.SplitN(raw, ":", 2)
		txID := ""
		if len(parts) == 2 {
			txID = strings.TrimSpace(parts[1])
		}
		return NewEventMessage("cashier.plate_mismatch", map[string]any{"transaction_id": txID})
	}

	return NewEventMessage("legacy.raw", map[string]any{"raw": raw})
}
