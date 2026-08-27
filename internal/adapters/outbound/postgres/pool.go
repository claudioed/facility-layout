// Package postgres provides pgxpool-backed implementations of every
// outbound port, plus a golang-migrate runner for the SQL migrations in
// /migrations.
package postgres

import (
	"context"
	"log/slog"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens a connection pool against databaseURL, traced with
// OpenTelemetry: every query, batch, copy and connection acquisition
// becomes a child span of whatever request or use case triggered it, so a
// slow endpoint can be attributed to a specific statement.
//
// otelpgx records the SQL statement in normalized form by default (no
// literal values), which is what keeps span attributes free of customer
// data and of unbounded cardinality — do not disable it.
//
// Pool statistics (idle/acquired/max connections) are also exported as
// metrics. Failing to register them is not fatal: telemetry never keeps the
// service from opening its database.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := otelpgx.RecordStats(pool); err != nil {
		slog.Warn("pgx pool stats metrics not registered", "error", err)
	}
	return pool, nil
}
