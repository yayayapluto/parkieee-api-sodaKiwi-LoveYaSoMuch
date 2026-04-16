package applog

import (
	"context"
	"errors"
	"log/slog"
)

// MultiHandler fans one log record out to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	out := make([]slog.Handler, 0, len(handlers))
	for _, h := range handlers {
		if h != nil {
			out = append(out, h)
		}
	}
	return &MultiHandler{handlers: out}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, child := range h.handlers {
		if child.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, rec slog.Record) error {
	var err error
	for _, child := range h.handlers {
		if !child.Enabled(ctx, rec.Level) {
			continue
		}
		if e := child.Handle(ctx, rec.Clone()); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	children := make([]slog.Handler, 0, len(h.handlers))
	for _, child := range h.handlers {
		children = append(children, child.WithAttrs(attrs))
	}
	return &MultiHandler{handlers: children}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	children := make([]slog.Handler, 0, len(h.handlers))
	for _, child := range h.handlers {
		children = append(children, child.WithGroup(name))
	}
	return &MultiHandler{handlers: children}
}
