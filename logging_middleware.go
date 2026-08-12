package logger

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record the start time
		start := time.Now()

		// Generate a new UUID v4 trace_id
		traceID := uuid.New().String()

		// Store trace_id in context
		ctx := WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		// Set X-Trace-Id response header before handler executes
		c.Header("X-Trace-Id", traceID)

		// Process request
		c.Next()

		// Calculate latency truncated to whole milliseconds
		latencyMs := time.Since(start).Milliseconds()

		// Build structured log attributes
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("latency_ms", latencyMs),
			slog.String("client_ip", c.ClientIP()),
			slog.String("trace_id", traceID),
		}

		// Use ErrorContext for status >= 500, InfoContext otherwise
		if c.Writer.Status() >= 500 {
			slog.LogAttrs(ctx, slog.LevelError, "HTTP request completed", attrs...)
		} else {
			slog.LogAttrs(ctx, slog.LevelInfo, "HTTP request completed", attrs...)
		}
	}
}
