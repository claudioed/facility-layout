package analyticsstore

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/claudioed/facility-layout/internal/analytics/report"
)

// PostgresProjection is the WRITER implementation of report.ProjectionStore,
// backed by a pgxpool over the analytical database. Every Apply* runs in a
// transaction that first claims the event id in analytics_processed_events
// (ON CONFLICT DO NOTHING); it only mutates the rollup when the claim is new,
// making each apply idempotent per eventId under Kafka's at-least-once
// delivery. It is the only writer of the analytical database.
type PostgresProjection struct {
	pool *pgxpool.Pool
}

// NewPostgresProjection constructs a PostgresProjection over pool.
func NewPostgresProjection(pool *pgxpool.Pool) *PostgresProjection {
	return &PostgresProjection{pool: pool}
}

// claim inserts eventId into analytics_processed_events, returning true iff this
// call newly recorded it (so the caller should apply the effect). It runs inside
// tx so the claim and the effect commit atomically.
func claim(ctx context.Context, tx pgx.Tx, eventId string, occurredAt time.Time) (bool, error) {
	tag, err := tx.Exec(ctx,
		`INSERT INTO analytics_processed_events (event_id, occurred_at)
		 VALUES ($1, $2) ON CONFLICT (event_id) DO NOTHING`,
		eventId, occurredAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// inTx runs fn in a transaction, committing on success and rolling back on
// error.
func (p *PostgresProjection) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// rollupDelta is the set of counter increments a single event contributes to a
// catalog-growth row.
type rollupDelta struct {
	sitesRegistered         int
	zonesRegistered         int
	aislesRegistered        int
	locationTypesRegistered int
	placementRulesDefined   int
	slotsRegistered         int
	slotsDecommissioned     int
	bulkImports             int
	importRowsSubmitted     int
	importRowsImported      int
	importRowsRejected      int
}

// apply claims eventId and, when the claim is new, upserts delta into the
// (scope, day_bucket) row. It is the shared body of every Apply* method.
func (p *PostgresProjection) apply(ctx context.Context, eventId, scope string, at time.Time, delta rollupDelta) error {
	return p.inTx(ctx, func(tx pgx.Tx) error {
		isNew, err := claim(ctx, tx, eventId, at)
		if err != nil {
			return fmt.Errorf("analyticsstore: claim event: %w", err)
		}
		if !isNew {
			return nil
		}
		return upsertRollup(ctx, tx, scope, at, delta)
	})
}

// ApplySiteRegistered records a site registration. Idempotent on eventId.
func (p *PostgresProjection) ApplySiteRegistered(ctx context.Context, eventId, scope string, at time.Time) error {
	return p.apply(ctx, eventId, scope, at, rollupDelta{sitesRegistered: 1})
}

// ApplyZoneRegistered records a zone registration. Idempotent on eventId.
func (p *PostgresProjection) ApplyZoneRegistered(ctx context.Context, eventId, scope string, at time.Time) error {
	return p.apply(ctx, eventId, scope, at, rollupDelta{zonesRegistered: 1})
}

// ApplyAisleRegistered records an aisle registration. Idempotent on eventId.
func (p *PostgresProjection) ApplyAisleRegistered(ctx context.Context, eventId, scope string, at time.Time) error {
	return p.apply(ctx, eventId, scope, at, rollupDelta{aislesRegistered: 1})
}

// ApplyLocationTypeRegistered records a location-type registration. Idempotent
// on eventId.
func (p *PostgresProjection) ApplyLocationTypeRegistered(ctx context.Context, eventId, scope string, at time.Time) error {
	return p.apply(ctx, eventId, scope, at, rollupDelta{locationTypesRegistered: 1})
}

// ApplyPlacementRuleDefined records a placement-rule definition. Idempotent on
// eventId.
func (p *PostgresProjection) ApplyPlacementRuleDefined(ctx context.Context, eventId, scope string, at time.Time) error {
	return p.apply(ctx, eventId, scope, at, rollupDelta{placementRulesDefined: 1})
}

// ApplyLocationSlotRegistered records a slot registration. Idempotent on
// eventId.
func (p *PostgresProjection) ApplyLocationSlotRegistered(ctx context.Context, eventId, scope string, at time.Time) error {
	return p.apply(ctx, eventId, scope, at, rollupDelta{slotsRegistered: 1})
}

// ApplyLocationSlotDecommissioned records a slot decommission. Idempotent on
// eventId.
func (p *PostgresProjection) ApplyLocationSlotDecommissioned(ctx context.Context, eventId, scope string, at time.Time) error {
	return p.apply(ctx, eventId, scope, at, rollupDelta{slotsDecommissioned: 1})
}

// ApplyFacilityLayoutImported records one bulk import and folds its row tallies
// into the scope's day bucket. Idempotent on eventId.
func (p *PostgresProjection) ApplyFacilityLayoutImported(ctx context.Context, eventId, scope string, submitted, imported, rejected int, at time.Time) error {
	return p.apply(ctx, eventId, scope, at, rollupDelta{
		bulkImports:         1,
		importRowsSubmitted: submitted,
		importRowsImported:  imported,
		importRowsRejected:  rejected,
	})
}

// upsertRollup adds delta into the (scope, day_bucket) row, inserting it if
// absent. day_bucket is derived by truncating at to the UTC day.
func upsertRollup(ctx context.Context, tx pgx.Tx, scope string, at time.Time, delta rollupDelta) error {
	bucket := at.UTC().Truncate(24 * time.Hour)
	_, err := tx.Exec(ctx,
		`INSERT INTO catalog_growth_rollup (
			scope, day_bucket,
			sites_registered, zones_registered, aisles_registered,
			location_types_registered, placement_rules_defined,
			slots_registered, slots_decommissioned,
			bulk_imports, import_rows_submitted, import_rows_imported, import_rows_rejected)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 ON CONFLICT (scope, day_bucket) DO UPDATE SET
			sites_registered          = catalog_growth_rollup.sites_registered + EXCLUDED.sites_registered,
			zones_registered          = catalog_growth_rollup.zones_registered + EXCLUDED.zones_registered,
			aisles_registered         = catalog_growth_rollup.aisles_registered + EXCLUDED.aisles_registered,
			location_types_registered = catalog_growth_rollup.location_types_registered + EXCLUDED.location_types_registered,
			placement_rules_defined   = catalog_growth_rollup.placement_rules_defined + EXCLUDED.placement_rules_defined,
			slots_registered          = catalog_growth_rollup.slots_registered + EXCLUDED.slots_registered,
			slots_decommissioned      = catalog_growth_rollup.slots_decommissioned + EXCLUDED.slots_decommissioned,
			bulk_imports              = catalog_growth_rollup.bulk_imports + EXCLUDED.bulk_imports,
			import_rows_submitted     = catalog_growth_rollup.import_rows_submitted + EXCLUDED.import_rows_submitted,
			import_rows_imported      = catalog_growth_rollup.import_rows_imported + EXCLUDED.import_rows_imported,
			import_rows_rejected      = catalog_growth_rollup.import_rows_rejected + EXCLUDED.import_rows_rejected`,
		scope, bucket,
		delta.sitesRegistered, delta.zonesRegistered, delta.aislesRegistered,
		delta.locationTypesRegistered, delta.placementRulesDefined,
		delta.slotsRegistered, delta.slotsDecommissioned,
		delta.bulkImports, delta.importRowsSubmitted, delta.importRowsImported, delta.importRowsRejected)
	if err != nil {
		return fmt.Errorf("analyticsstore: upsert rollup: %w", err)
	}
	return nil
}

// Compile-time assertion that PostgresProjection satisfies the write port.
var _ report.ProjectionStore = (*PostgresProjection)(nil)
