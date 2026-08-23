# Structured logging

facility-layout logs with Go's standard library `log/slog`, emitting JSON to
stdout. There is no third-party logging dependency.

## Why `log/slog`

- **Zero new dependency.** `log/slog` shipped in Go 1.21 and this service is
  already on a current toolchain; pulling in zerolog or zap would add
  dependency weight for capability the standard library already provides.
- **Ecosystem standard.** `log/slog` is the shape the wider Go ecosystem
  (including `net/http`, `database/sql` drivers, and most middleware) is
  converging on for structured, leveled logging.
- **OTel-bridge-ready.** `log/slog`'s `Handler` interface is exactly what the
  OpenTelemetry Go SDK's log bridge (`go.opentelemetry.io/contrib/bridges/otelslog`)
  expects, so wiring this service into an OTel collector later is a handler
  swap, not a rewrite.

This mirrors the logging decision made across the sibling warehouse-systems
services.

## Configuration

| Env var     | Values                              | Default | Effect                       |
|-------------|--------------------------------------|---------|-------------------------------|
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` (case-insensitive) | `info`  | Minimum level emitted |

Unrecognized values fall back to `info`.

## Output shape

Every log line is a single JSON object written to stdout via
`slog.NewJSONHandler`, e.g.:

```json
{"time":"2026-08-23T12:00:00Z","level":"INFO","msg":"http server listening","addr":":8080"}
{"time":"2026-08-23T12:00:01Z","level":"INFO","msg":"http request","method":"POST","path":"/sites","status":201,"duration_ms":4,"bytes":128,"request_id":"abc123"}
{"time":"2026-08-23T12:00:01Z","level":"INFO","msg":"domain event published","event_name":"SiteRegistered","event_type":"com.claudioed.facility-layout.site.registered","payload":{"...":"..."}}
```

## Where it's wired in

- `cmd/facility/main.go` builds the process-wide `*slog.Logger` via
  `newLogger(LOG_LEVEL)` as the first step of `run()` and calls
  `slog.SetDefault(logger)` so any stdlib/third-party code that logs through
  `slog.Default()` picks it up too.
- `internal/adapters/inbound/http/logging.go` provides `RequestLogger`, chi
  middleware that logs method, path, status, duration, response size, and
  the chi request ID for every HTTP request (`Error` level for 5xx
  responses, `Info` otherwise). It is installed in `NewRouter` right after
  `middleware.RequestID` and before `middleware.Recoverer`.
- `internal/adapters/outbound/events/log_publisher.go`'s `LogPublisher`
  (the default `EventPublisher` when no database is configured) logs each
  published domain event as a structured `slog.Info` call instead of a
  formatted string.
