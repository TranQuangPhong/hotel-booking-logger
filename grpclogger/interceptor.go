// Package grpclogger is the gRPC side of logger, kept apart so HTTP-only services don't pull in grpc.
package grpclogger

import (
	"context"
	"log/slog"
	"net"
	"time"

	logger "github.com/TranQuangPhong/hotel-booking-logger"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// traceIDKey is the gRPC metadata key for trace_id (metadata keys are lowercase).
const traceIDKey = "x-trace-id"

// UnaryServerInterceptor logs one line per unary gRPC call, the gRPC counterpart of logger.LoggingMiddleware.
// Chain it before recovery so a recovered panic is logged as Internal.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Record the start time
		start := time.Now()

		// Reuse the caller's trace_id from metadata, or generate a new UUID v4 one
		traceID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get(traceIDKey); len(v) > 0 {
				traceID = v[0]
			}
		}
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Store trace_id in context
		ctx = logger.WithTraceID(ctx, traceID)

		// Send x-trace-id as a response header; it fails only if headers were already sent
		_ = grpc.SetHeader(ctx, metadata.Pairs(traceIDKey, traceID))

		// Process request
		resp, err := handler(ctx, req)

		// Build structured log attributes; logger.TraceHandler adds trace_id from ctx
		code := status.Code(err)
		attrs := []slog.Attr{
			slog.String("method", info.FullMethod),
			slog.String("code", code.String()),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
		}
		if p, ok := peer.FromContext(ctx); ok {
			// IP only, like Gin's ClientIP(); a non-TCP peer (e.g. bufconn in tests) keeps its full address
			clientIP := p.Addr.String()
			if tcp, ok := p.Addr.(*net.TCPAddr); ok {
				clientIP = tcp.IP.String()
			}
			attrs = append(attrs, slog.String("client_ip", clientIP))
		}
		// Every generated request with a saga_id field has this getter
		if r, ok := req.(interface{ GetSagaId() string }); ok {
			attrs = append(attrs, slog.String("saga_id", r.GetSagaId()))
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", status.Convert(err).Message()))
		}

		// Use LevelError for server faults (the HTTP 5xx of gRPC), LevelInfo otherwise
		switch code {
		case codes.Unknown, codes.Internal, codes.DataLoss, codes.Unimplemented:
			slog.LogAttrs(ctx, slog.LevelError, "gRPC request completed", attrs...)
		default:
			slog.LogAttrs(ctx, slog.LevelInfo, "gRPC request completed", attrs...)
		}
		return resp, err
	}
}
