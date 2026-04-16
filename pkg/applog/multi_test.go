package applog

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureHandler struct {
	enabled   bool
	handled   int
	lastAttrs []slog.Attr
	group     string
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return h.enabled }

func (h *captureHandler) Handle(_ context.Context, rec slog.Record) error {
	h.handled++
	h.lastAttrs = h.lastAttrs[:0]
	rec.Attrs(func(a slog.Attr) bool {
		h.lastAttrs = append(h.lastAttrs, a)
		return true
	})
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.lastAttrs = append(append([]slog.Attr(nil), h.lastAttrs...), attrs...)
	return &clone
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.group = name
	return &clone
}

func TestMultiHandler_FansOutToAllEnabledHandlers(t *testing.T) {
	h1 := &captureHandler{enabled: true}
	h2 := &captureHandler{enabled: true}

	multi := NewMultiHandler(h1, h2)
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	rec.AddAttrs(slog.String("key", "value"))

	require.NoError(t, multi.Handle(context.Background(), rec))
	assert.Equal(t, 1, h1.handled)
	assert.Equal(t, 1, h2.handled)
}

func TestMultiHandler_EnabledReflectsChildren(t *testing.T) {
	allOff := NewMultiHandler(&captureHandler{enabled: false}, &captureHandler{enabled: false})
	assert.False(t, allOff.Enabled(context.Background(), slog.LevelInfo))

	oneOn := NewMultiHandler(&captureHandler{enabled: false}, &captureHandler{enabled: true})
	assert.True(t, oneOn.Enabled(context.Background(), slog.LevelInfo))
}
