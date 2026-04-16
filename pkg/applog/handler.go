package applog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
)

const defaultBufferSize = 500

type insertFunc func(context.Context, *AppLog) error

type dbHandlerOptions struct {
	bufferSize  int
	startWorker bool
}

type queuedRecord struct {
	time    time.Time
	level   slog.Level
	message string
	attrs   []slog.Attr
	groups  []string
}

// DBHandler is a slog handler that asynchronously persists logs to app_logs.
type DBHandler struct {
	queue  chan queuedRecord
	insert insertFunc
	attrs  []slog.Attr
	groups []string
}

// NewDBHandler creates a DB-backed slog handler with a buffered queue and one drain goroutine.
func NewDBHandler(db *gorm.DB) *DBHandler {
	ins := func(ctx context.Context, row *AppLog) error {
		return db.WithContext(ctx).Create(row).Error
	}
	return newDBHandler(ins, dbHandlerOptions{bufferSize: defaultBufferSize, startWorker: true})
}

func newDBHandler(ins insertFunc, opts dbHandlerOptions) *DBHandler {
	if opts.bufferSize <= 0 {
		opts.bufferSize = defaultBufferSize
	}
	h := &DBHandler{
		queue:  make(chan queuedRecord, opts.bufferSize),
		insert: ins,
		attrs:  make([]slog.Attr, 0),
		groups: make([]string, 0),
	}
	if opts.startWorker {
		go h.drain()
	}
	return h
}

func (h *DBHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *DBHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+record.NumAttrs())
	copy(attrs, h.attrs)
	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	rec := queuedRecord{
		time:    record.Time,
		level:   record.Level,
		message: record.Message,
		attrs:   attrs,
		groups:  append([]string(nil), h.groups...),
	}

	select {
	case h.queue <- rec:
	default:
		// Avoid slog usage here so we do not recursively log into the same full queue.
		_, _ = fmt.Fprintln(os.Stderr, "applog: buffer full, dropping log record")
	}

	return nil
}

func (h *DBHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copyAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(copyAttrs, h.attrs)
	copyAttrs = append(copyAttrs, attrs...)

	return &DBHandler{
		queue:  h.queue,
		insert: h.insert,
		attrs:  copyAttrs,
		groups: append([]string(nil), h.groups...),
	}
}

func (h *DBHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	copyGroups := append([]string(nil), h.groups...)
	copyGroups = append(copyGroups, name)

	return &DBHandler{
		queue:  h.queue,
		insert: h.insert,
		attrs:  append([]slog.Attr(nil), h.attrs...),
		groups: copyGroups,
	}
}

func (h *DBHandler) drain() {
	for rec := range h.queue {
		row, err := buildRow(rec)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "applog: marshal failed: %v\n", err)
			continue
		}
		if err := h.insert(context.Background(), row); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "applog: insert failed: %v\n", err)
		}
	}
}

func buildRow(rec queuedRecord) (*AppLog, error) {
	fields := make(map[string]any)
	var source string
	var requestID *string

	for _, attr := range rec.attrs {
		flattenAttr(fields, rec.groups, attr)
	}

	if val, ok := fields[slog.SourceKey]; ok {
		source = sourceFromAny(val)
	}

	if val, ok := fields["request_id"]; ok {
		if v, ok := val.(string); ok && v != "" {
			requestID = &v
		}
	}

	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	createdAt := rec.time
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return &AppLog{
		Level:     strings.ToUpper(rec.level.String()),
		Source:    source,
		Message:   rec.message,
		Fields:    encoded,
		RequestID: requestID,
		CreatedAt: createdAt,
	}, nil
}

func flattenAttr(fields map[string]any, groups []string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		nextGroups := groups
		if attr.Key != "" {
			nextGroups = append(append([]string(nil), groups...), attr.Key)
		}
		for _, nested := range attr.Value.Group() {
			flattenAttr(fields, nextGroups, nested)
		}
		return
	}

	keyParts := append([]string(nil), groups...)
	if attr.Key != "" {
		keyParts = append(keyParts, attr.Key)
	}
	if len(keyParts) == 0 {
		return
	}

	key := strings.Join(keyParts, ".")
	fields[key] = valueToAny(attr.Value)
}

func valueToAny(v slog.Value) any {
	v = v.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindBool:
		return v.Bool()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339Nano)
	case slog.KindGroup:
		out := make(map[string]any)
		for _, nested := range v.Group() {
			flattenAttr(out, nil, nested)
		}
		return out
	case slog.KindLogValuer:
		return valueToAny(v.LogValuer().LogValue())
	default:
		return v.Any()
	}
}

func sourceFromAny(v any) string {
	if src, ok := v.(*slog.Source); ok && src != nil {
		if src.Function != "" {
			return src.Function
		}
		return src.File
	}
	if src, ok := v.(slog.Source); ok {
		if src.Function != "" {
			return src.Function
		}
		return src.File
	}
	if src, ok := v.(map[string]any); ok {
		if fn, ok := src["function"].(string); ok && fn != "" {
			return fn
		}
		if file, ok := src["file"].(string); ok {
			return file
		}
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
