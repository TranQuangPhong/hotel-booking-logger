package logger

import "context"

// contextKey is an unexported type to prevent collisions with other packages.
type contextKey struct{}

// WithTraceID stores a trace_id in the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, contextKey{}, traceID)
}

// GetTraceID retrieves the trace_id from the context.
// Returns empty string if no trace_id is present.
func GetTraceID(ctx context.Context) string {
	v, ok := ctx.Value(contextKey{}).(string)
	if !ok {
		return ""
	}
	return v
}
