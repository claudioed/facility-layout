-- Facility-layout "Layout Catalog Growth & Change" analytics read model
-- (ADR-0010).
--
-- This is the ANALYTICAL database, separate from the OLTP database. It is
-- written only by cmd/facility-projector and read (read-only) by
-- cmd/facility-reports. The tables here are projections derived from the
-- analytics event stream, not sources of truth.

-- Idempotency + freshness: every applied analytics event id is recorded here
-- exactly once. applied_at is wall-clock insert time; occurred_at is the
-- event's business time, used to compute the projection's freshness lag.
CREATE TABLE analytics_processed_events (
    event_id    TEXT PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_analytics_processed_events_occurred_at
    ON analytics_processed_events (occurred_at DESC);

-- Consumer-level dedupe set, used by the inbound consumer's idempotency gate.
-- It is kept SEPARATE from analytics_processed_events (which the projection
-- UPSERT claims) so the two idempotency layers do not race to claim the same
-- event_id: the consumer gate admits the event, the projection then records
-- its effect.
CREATE TABLE analytics_consumed_events (
    event_id     TEXT PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The catalog-growth rollup fact table: one row per (scope, day_bucket).
-- `scope` is a site code or zone id, or the empty string for catalog-wide
-- definitions (location types, placement rules, bulk imports). Counters are
-- UPSERTed as events arrive. Because the warehouse map is a slow-changing
-- reference catalog, rows are bucketed by DAY (midnight UTC), not by hour.
CREATE TABLE catalog_growth_rollup (
    scope                     TEXT NOT NULL,
    day_bucket                TIMESTAMPTZ NOT NULL,
    sites_registered          BIGINT NOT NULL DEFAULT 0,
    zones_registered          BIGINT NOT NULL DEFAULT 0,
    aisles_registered         BIGINT NOT NULL DEFAULT 0,
    location_types_registered BIGINT NOT NULL DEFAULT 0,
    placement_rules_defined   BIGINT NOT NULL DEFAULT 0,
    slots_registered          BIGINT NOT NULL DEFAULT 0,
    slots_decommissioned      BIGINT NOT NULL DEFAULT 0,
    bulk_imports              BIGINT NOT NULL DEFAULT 0,
    import_rows_submitted     BIGINT NOT NULL DEFAULT 0,
    import_rows_imported      BIGINT NOT NULL DEFAULT 0,
    import_rows_rejected      BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (scope, day_bucket)
);

CREATE INDEX idx_catalog_growth_rollup_day_bucket
    ON catalog_growth_rollup (day_bucket);
