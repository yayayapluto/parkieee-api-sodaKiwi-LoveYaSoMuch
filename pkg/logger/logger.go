// pkg/logger/logger.go
package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/yyypluto/parkieee-api/pkg/applog"
	"gorm.io/gorm"
)

type contextKey string

const (
	keyRequestID contextKey = "request_id"
	keyUserID    contextKey = "user_id"
)

var global *slog.Logger

// Init initialises the global logger. Call once at startup before serving requests.
// level:  "debug" | "info" (default) | "warn" | "error"
// format: "json" (production) | "text" (development, default)
func Init(level, format string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	global = slog.New(handler)
	slog.SetDefault(global)
}

// AddDBHandler fans out slog output to both the existing handler and app_logs in PostgreSQL.
func AddDBHandler(db *gorm.DB) {
	if db == nil {
		return
	}
	if global == nil {
		Init("info", "text")
	}
	global = slog.New(applog.NewMultiHandler(global.Handler(), applog.NewDBHandler(db)))
	slog.SetDefault(global)
}

// Get returns the global logger. Falls back to the default slog logger if Init was not called.
func Get() *slog.Logger {
	if global == nil {
		return slog.Default()
	}
	return global
}

// WithRequestID returns a child logger with the request_id field pre-attached.
func WithRequestID(requestID string) *slog.Logger {
	return Get().With("request_id", requestID)
}

// ContextWithRequestID stores a request ID in the context for downstream retrieval.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, keyRequestID, requestID)
}

// ContextWithUserID stores a user ID in the context for downstream retrieval.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, keyUserID, userID)
}

// FromContext returns a logger enriched with any request_id / user_id stored in ctx.
func FromContext(ctx context.Context) *slog.Logger {
	l := Get()
	if v, ok := ctx.Value(keyRequestID).(string); ok && v != "" {
		l = l.With("request_id", v)
	}
	if v, ok := ctx.Value(keyUserID).(string); ok && v != "" {
		l = l.With("user_id", v)
	}
	return l
}
