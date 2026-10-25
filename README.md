# hotel-booking-logger

Shared structured logging for the hotel-booking services: `log/slog`, JSON to stdout, a `trace_id` on every line.

| Import | Gives you |
|---|---|
| `github.com/TranQuangPhong/hotel-booking-logger` (`logger`) | `NewLogger()`, Gin `LoggingMiddleware()`, `WithTraceID` / `GetTraceID` |
| `github.com/TranQuangPhong/hotel-booking-logger/grpclogger` | `UnaryServerInterceptor()`. Import it only in services that serve gRPC, so HTTP-only ones don't build grpc |

## Usage

```go
slog.SetDefault(logger.NewLogger())

r := gin.New()
r.Use(gin.Recovery())
r.Use(logger.LoggingMiddleware())

grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
	grpclogger.UnaryServerInterceptor(), // first = outermost, so a recovered panic is logged as Internal
	recovery.UnaryServerInterceptor(),
))
```

Log with the request context (`slog.InfoContext(ctx, ...)`, not `slog.Info(...)`), otherwise the line has no `trace_id`.

## What gets logged

- **Level:** `LOG_LEVEL` = `debug` | `info` (default) | `warn` | `error`, case-insensitive. An invalid value falls back to `info` with a warning.
- **trace_id:** `TraceHandler` reads it from the context and puts it first on every line.
- **HTTP:** one `HTTP request completed` line per request: `method`, `path`, `status`, `latency_ms`, `client_ip`. Each request gets a new trace_id, returned in the `X-Trace-Id` header. Level is Error for status >= 500, Info otherwise.
- **gRPC:** one `gRPC request completed` line per unary call: `method`, `code`, `latency_ms`, `client_ip`, plus `saga_id` when the request has that field and `error` when the call fails. The trace_id comes from the incoming `x-trace-id` metadata, or is a new UUID, and is returned in the `x-trace-id` response header. Level is Error for `Unknown`, `Internal`, `DataLoss`, `Unimplemented`, Info otherwise.

## Release

Tag a new version and push it, then update each service: `go get github.com/TranQuangPhong/hotel-booking-logger@vX.Y.Z`.
