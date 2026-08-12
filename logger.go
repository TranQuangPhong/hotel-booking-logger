package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// TraceHandler wraps an slog.Handler to automatically inject trace_id from context.
type TraceHandler struct {
	inner slog.Handler
}

// NewTraceHandler creates a TraceHandler wrapping the given handler.
func NewTraceHandler(hander slog.Handler) *TraceHandler {
	return &TraceHandler{inner: hander}
}

// Enabled delegates to the inner handler.
func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle extracts trace_id from ctx and prepends it to the record's attributes.
// If trace_id is non-empty, it adds it as the first attribute.
// It also removes any manually-added "trace_id" attribute to ensure context value takes precedence.
func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	traceID := GetTraceID(ctx)
	if traceID == "" {
		return h.inner.Handle(ctx, r)
	}

	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != "trace_id" {
			attrs = append(attrs, a)
		}
		return true
	})

	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	newRecord.AddAttrs(slog.String("trace_id", traceID))
	newRecord.AddAttrs(attrs...)

	return h.inner.Handle(ctx, newRecord)
}

// WithAttrs delegates to the inner handler, returning a new TraceHandler.
func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup delegates to the inner handler, returning a new TraceHandler.
func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{inner: h.inner.WithGroup(name)}
}

// NewLogger creates a configured *slog.Logger with TraceHandler wrapping JSONHandler.
// Reads LOG_LEVEL env var (case-insensitive: Debug, Info, Warn, Error).
// Falls back to slog.LevelInfo when missing or invalid.
// Emits a warning log if the value is present but invalid.
func NewLogger() *slog.Logger {
	level := slog.LevelInfo
	invalidLevel := ""

	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch strings.ToLower(envLevel) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			invalidLevel = envLevel
		}
	}
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})
	traceHandler := NewTraceHandler(jsonHandler)
	logger := slog.New(traceHandler)

	if invalidLevel != "" {
		logger.Warn("Invalid LOG_LEVEL value, falling back to Info",
			slog.String("invalid_value", invalidLevel),
			slog.String("fallback", "info"),
		)
	}
	return logger
}
