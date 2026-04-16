package applog

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBHandler_EnabledAlwaysTrue(t *testing.T) {
	h := newDBHandler(func(context.Context, *AppLog) error { return nil }, dbHandlerOptions{bufferSize: 1, startWorker: false})
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
	assert.True(t, h.Enabled(context.Background(), slog.LevelDebug))
}

func TestDBHandler_PersistsStructuredRecord(t *testing.T) {
	rows := make(chan *AppLog, 1)
	h := newDBHandler(func(_ context.Context, row *AppLog) error {
		rows <- row
		return nil
	}, dbHandlerOptions{bufferSize: 2, startWorker: true})

	rec := slog.NewRecord(time.Now(), slog.LevelWarn, "payment warning", 0)
	rec.AddAttrs(
		slog.String("request_id", "req-123"),
		slog.String("method", "cash"),
		slog.Int("status", 409),
	)

	require.NoError(t, h.Handle(context.Background(), rec))

	select {
	case row := <-rows:
		assert.Equal(t, "WARN", row.Level)
		assert.Equal(t, "payment warning", row.Message)
		require.NotNil(t, row.RequestID)
		assert.Equal(t, "req-123", *row.RequestID)

		fields := map[string]any{}
		require.NoError(t, json.Unmarshal(row.Fields, &fields))
		assert.Equal(t, "cash", fields["method"])
		assert.EqualValues(t, 409, fields["status"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for db insert")
	}
}

func TestDBHandler_DropsWhenBufferFull(t *testing.T) {
	h := newDBHandler(func(context.Context, *AppLog) error { return nil }, dbHandlerOptions{bufferSize: 1, startWorker: false})

	first := slog.NewRecord(time.Now(), slog.LevelInfo, "first", 0)
	second := slog.NewRecord(time.Now(), slog.LevelInfo, "second", 0)

	require.NoError(t, h.Handle(context.Background(), first))
	require.NoError(t, h.Handle(context.Background(), second))

	// Buffer stays full with only the first record because second is dropped.
	assert.Equal(t, 1, len(h.queue))

	queued := <-h.queue
	assert.Equal(t, "first", queued.message)
}
