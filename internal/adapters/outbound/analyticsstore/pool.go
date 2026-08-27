package analyticsstore

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool builds a pgxpool over the analytical database at databaseURL. It is
// used by the writer (cmd/facility-projector). Unlike the OLTP postgres.NewPool
// this pool is untraced: facility-layout's analytics processes are kept free of
// the OTel pgx tracer, consistent with the trace-free analytics publisher.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

// NewReadOnlyPool builds a pgxpool over the analytical database in which every
// connection is pinned to a read-only transaction default
// (default_transaction_read_only=on). The reader process
// (cmd/facility-reports) uses this so a bug there cannot mutate the read model
// even if the database role itself is not read-only — defence in depth on top
// of the read-only ANALYTICS_DATABASE_URL role (ADR-0010).
func NewReadOnlyPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	return pgxpool.NewWithConfig(ctx, cfg)
}
