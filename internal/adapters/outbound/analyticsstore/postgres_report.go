package analyticsstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/claudioed/facility-layout/internal/analytics/report"
)

// PostgresReport is the READER implementation of report.ReportStore, backed by
// a pgxpool over the analytical database. The pool it is given is expected to be
// pinned to a read-only role / default_transaction_read_only=on, so a bug in the
// reader cannot mutate the read model (ADR-0010). The reader never issues
// writes.
type PostgresReport struct {
	pool *pgxpool.Pool
}

// NewPostgresReport constructs a PostgresReport over pool.
func NewPostgresReport(pool *pgxpool.Pool) *PostgresReport {
	return &PostgresReport{pool: pool}
}

// Query returns the catalog-growth rows matching q. From is inclusive, To is
// exclusive; an empty Scope disables the scope filter.
func (r *PostgresReport) Query(ctx context.Context, q report.ReportQuery) (report.CatalogReport, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT scope, day_bucket,
			sites_registered, zones_registered, aisles_registered,
			location_types_registered, placement_rules_defined,
			slots_registered, slots_decommissioned,
			bulk_imports, import_rows_submitted, import_rows_imported, import_rows_rejected
		 FROM catalog_growth_rollup
		 WHERE day_bucket >= $1 AND day_bucket < $2
		   AND ($3 = '' OR scope = $3)
		 ORDER BY day_bucket, scope`,
		q.From, q.To, q.Scope)
	if err != nil {
		return report.CatalogReport{}, fmt.Errorf("analyticsstore: query rollup: %w", err)
	}
	defer rows.Close()

	var out report.CatalogReport
	for rows.Next() {
		var (
			row    report.Row
			bucket time.Time
		)
		if err := rows.Scan(
			&row.Key.Scope, &bucket,
			&row.SitesRegistered, &row.ZonesRegistered, &row.AislesRegistered,
			&row.LocationTypesRegistered, &row.PlacementRulesDefined,
			&row.SlotsRegistered, &row.SlotsDecommissioned,
			&row.BulkImports, &row.ImportRowsSubmitted, &row.ImportRowsImported, &row.ImportRowsRejected,
		); err != nil {
			return report.CatalogReport{}, fmt.Errorf("analyticsstore: scan row: %w", err)
		}
		row.Key.DayBucket = bucket.UTC()
		out.Rows = append(out.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return report.CatalogReport{}, fmt.Errorf("analyticsstore: iterate rows: %w", err)
	}
	return out, nil
}

// FreshnessLag returns now minus the most recent event's occurred_at, i.e. how
// far the read model trails real time. Zero when the read model is empty or
// (defensively) when the latest event is future-dated.
func (r *PostgresReport) FreshnessLag(ctx context.Context) (time.Duration, error) {
	// max() over an empty table returns a single NULL row (not zero rows), so
	// scan into a nullable *time.Time and treat NULL as "read model empty".
	var latest *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT max(occurred_at) FROM analytics_processed_events`).Scan(&latest)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("analyticsstore: freshness query: %w", err)
	}
	if latest == nil || latest.IsZero() {
		return 0, nil
	}
	lag := time.Since(*latest)
	if lag < 0 {
		return 0, nil
	}
	return lag, nil
}

// Compile-time assertion that PostgresReport satisfies the read port.
var _ report.ReportStore = (*PostgresReport)(nil)
