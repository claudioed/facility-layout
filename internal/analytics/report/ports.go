package report

import (
	"context"
	"time"
)

// ReportStore is the read side of the catalog-growth data product: the reader
// process queries it to serve reports. It is read-only by contract — the
// Postgres implementation runs over a pool pinned to a read-only role.
type ReportStore interface {
	// Query returns the catalog-growth rows matching q.
	Query(ctx context.Context, q ReportQuery) (CatalogReport, error)
	// FreshnessLag reports how far the read model lags real time: the age of
	// the most recently applied event. A larger lag means the projection is
	// further behind the event stream.
	FreshnessLag(ctx context.Context) (time.Duration, error)
}

// ProjectionStore is the write side of the catalog-growth data product: the
// projector process applies each consumed event to it. Every Apply* method is
// idempotent on eventId — applying the same eventId twice records the effect
// once, so the at-least-once Kafka stream can be projected exactly once.
//
// The methods take the derivation-relevant fields already extracted from the
// analytics envelope (rather than a domain event) so this port stays free of
// any OLTP domain dependency. `scope` is the site code or zone id the change
// belongs to, or the empty string for a catalog-wide definition; `at` is the
// event's business time, from which the day bucket is derived.
type ProjectionStore interface {
	// ApplySiteRegistered records a site registration in scope's day bucket.
	ApplySiteRegistered(ctx context.Context, eventId, scope string, at time.Time) error
	// ApplyZoneRegistered records a zone registration.
	ApplyZoneRegistered(ctx context.Context, eventId, scope string, at time.Time) error
	// ApplyAisleRegistered records an aisle registration.
	ApplyAisleRegistered(ctx context.Context, eventId, scope string, at time.Time) error
	// ApplyLocationTypeRegistered records a location-type registration.
	ApplyLocationTypeRegistered(ctx context.Context, eventId, scope string, at time.Time) error
	// ApplyPlacementRuleDefined records a placement-rule definition.
	ApplyPlacementRuleDefined(ctx context.Context, eventId, scope string, at time.Time) error
	// ApplyLocationSlotRegistered records a slot registration.
	ApplyLocationSlotRegistered(ctx context.Context, eventId, scope string, at time.Time) error
	// ApplyLocationSlotDecommissioned records a slot decommission.
	ApplyLocationSlotDecommissioned(ctx context.Context, eventId, scope string, at time.Time) error
	// ApplyFacilityLayoutImported records one bulk import and folds its row
	// tallies (submitted, imported, rejected) into scope's day bucket.
	ApplyFacilityLayoutImported(ctx context.Context, eventId, scope string, submitted, imported, rejected int, at time.Time) error
}
