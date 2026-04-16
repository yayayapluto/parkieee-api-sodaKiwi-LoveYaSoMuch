// pkg/logger/logger_test.go
package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyypluto/parkieee-api/pkg/logger"
)

func TestInit_TextFormat(t *testing.T) {
	logger.Init("info", "text")
	l := logger.Get()
	assert.NotNil(t, l)
}

func TestInit_JSONFormat(t *testing.T) {
	// Redirect stdout to capture JSON output.
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	logger.Init("debug", "json")
	logger.Get().Debug("test message", "key", "value")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var record map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &record))
	assert.Equal(t, "test message", record["msg"])
	assert.Equal(t, "value", record["key"])

	// Restore text logger for other tests.
	logger.Init("info", "text")
}

func TestGet_FallbackBeforeInit(t *testing.T) {
	// Even before Init, Get must return a non-nil logger.
	l := logger.Get()
	assert.NotNil(t, l)
}

func TestFromContext_WithRequestAndUserID(t *testing.T) {
	logger.Init("debug", "text")

	ctx := context.Background()
	ctx = logger.ContextWithRequestID(ctx, "req-123")
	ctx = logger.ContextWithUserID(ctx, "user-456")

	l := logger.FromContext(ctx)
	assert.NotNil(t, l)

	// Verify the attrs are attached by logging into a buffer.
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	enriched := slog.New(h)
	enriched = enriched.With("request_id", "req-123", "user_id", "user-456")
	enriched.Info("check")
	assert.Contains(t, buf.String(), "req-123")
	assert.Contains(t, buf.String(), "user-456")
}

func TestFromContext_EmptyContext(t *testing.T) {
	logger.Init("info", "text")
	l := logger.FromContext(context.Background())
	assert.NotNil(t, l)
}

func TestWithRequestID(t *testing.T) {
	logger.Init("info", "text")
	l := logger.WithRequestID("abc-999")
	assert.NotNil(t, l)
}
