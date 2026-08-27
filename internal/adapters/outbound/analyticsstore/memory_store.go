// Package analyticsstore provides the outbound adapters that persist and serve
// the facility-layout "Layout Catalog Growth & Change" read model: an in-memory
// implementation (MemoryStore) for tests and local runs, and Postgres
// implementations (a writer projection and a read-only reader) for deployment.
// All satisfy the report.ProjectionStore and/or report.ReportStore ports.
package analyticsstore

import (
	"context"
	"sync"
	"time"

	"github.com/claudioed/facility-layout/internal/analytics/report"
)

// MemoryStore is an in-memory implementation of both report.ProjectionStore
// (write) and report.ReportStore (read), backed by maps. It is idempotent per
// eventId via a seen-set, so a duplicate delivery is a no-op. It is safe for
// concurrent use.
type MemoryStore struct {
	// Now supplies the current time for FreshnessLag; defaults to time.Now
	// when nil so lag is deterministic under test.
	Now func() time.Time

	mu   sync.Mutex
	seen map[string]struct{}
	rows map[report.RowKey]*report.Row
	// latest is the OccurredAt of the most recently applied event, used to
	// compute FreshnessLag.
	latest time.Time
}

// NewMemoryStore constructs an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		seen: map[string]struct{}{},
		rows: map[report.RowKey]*report.Row{},
	}
}

func dayBucket(t time.Time) time.Time { return t.UTC().Truncate(24 * time.Hour) }

// firstApply marks eventId as seen and reports whether this is the first time
// (so the caller should apply the effect) or a duplicate (skip). It also
// advances the freshness watermark. The caller must hold s.mu.
func (s *MemoryStore) firstApply(eventId string, at time.Time) bool {
	if _, dup := s.seen[eventId]; dup {
		return false
	}
	s.seen[eventId] = struct{}{}
	if at.After(s.latest) {
		s.latest = at
	}
	return true
}

func (s *MemoryStore) row(scope string, at time.Time) *report.Row {
	k := report.RowKey{Scope: scope, DayBucket: dayBucket(at)}
	r, ok := s.rows[k]
	if !ok {
		r = &report.Row{Key: k}
		s.rows[k] = r
	}
	return r
}

// mutate is the shared body of every counter-only Apply* method: dedupe, then
// apply fn to the scope's day-bucket row.
func (s *MemoryStore) mutate(eventId, scope string, at time.Time, fn func(*report.Row)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.firstApply(eventId, at) {
		return nil
	}
	fn(s.row(scope, at))
	return nil
}

// ApplySiteRegistered records a site registration. Idempotent on eventId.
func (s *MemoryStore) ApplySiteRegistered(_ context.Context, eventId, scope string, at time.Time) error {
	return s.mutate(eventId, scope, at, func(r *report.Row) { r.SitesRegistered++ })
}

// ApplyZoneRegistered records a zone registration. Idempotent on eventId.
func (s *MemoryStore) ApplyZoneRegistered(_ context.Context, eventId, scope string, at time.Time) error {
	return s.mutate(eventId, scope, at, func(r *report.Row) { r.ZonesRegistered++ })
}

// ApplyAisleRegistered records an aisle registration. Idempotent on eventId.
func (s *MemoryStore) ApplyAisleRegistered(_ context.Context, eventId, scope string, at time.Time) error {
	return s.mutate(eventId, scope, at, func(r *report.Row) { r.AislesRegistered++ })
}

// ApplyLocationTypeRegistered records a location-type registration. Idempotent
// on eventId.
func (s *MemoryStore) ApplyLocationTypeRegistered(_ context.Context, eventId, scope string, at time.Time) error {
	return s.mutate(eventId, scope, at, func(r *report.Row) { r.LocationTypesRegistered++ })
}

// ApplyPlacementRuleDefined records a placement-rule definition. Idempotent on
// eventId.
func (s *MemoryStore) ApplyPlacementRuleDefined(_ context.Context, eventId, scope string, at time.Time) error {
	return s.mutate(eventId, scope, at, func(r *report.Row) { r.PlacementRulesDefined++ })
}

// ApplyLocationSlotRegistered records a slot registration. Idempotent on
// eventId.
func (s *MemoryStore) ApplyLocationSlotRegistered(_ context.Context, eventId, scope string, at time.Time) error {
	return s.mutate(eventId, scope, at, func(r *report.Row) { r.SlotsRegistered++ })
}

// ApplyLocationSlotDecommissioned records a slot decommission. Idempotent on
// eventId.
func (s *MemoryStore) ApplyLocationSlotDecommissioned(_ context.Context, eventId, scope string, at time.Time) error {
	return s.mutate(eventId, scope, at, func(r *report.Row) { r.SlotsDecommissioned++ })
}

// ApplyFacilityLayoutImported records one bulk import and folds its row tallies
// into the scope's day bucket. Idempotent on eventId.
func (s *MemoryStore) ApplyFacilityLayoutImported(_ context.Context, eventId, scope string, submitted, imported, rejected int, at time.Time) error {
	return s.mutate(eventId, scope, at, func(r *report.Row) {
		r.BulkImports++
		r.ImportRowsSubmitted += submitted
		r.ImportRowsImported += imported
		r.ImportRowsRejected += rejected
	})
}

// Query returns the rows matching q. From is inclusive, To is exclusive, both
// compared against a row's DayBucket; an empty Scope means no filter on scope.
func (s *MemoryStore) Query(_ context.Context, q report.ReportQuery) (report.CatalogReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := report.CatalogReport{}
	for k, r := range s.rows {
		if k.DayBucket.Before(q.From) || !k.DayBucket.Before(q.To) {
			continue
		}
		if q.Scope != "" && k.Scope != q.Scope {
			continue
		}
		row := *r
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// FreshnessLag returns how far the read model lags real time: now minus the
// OccurredAt of the most recently applied event. Zero when nothing has been
// applied yet, and never negative (a future-dated event clamps to zero).
func (s *MemoryStore) FreshnessLag(_ context.Context) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.latest.IsZero() {
		return 0, nil
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	lag := now.Sub(s.latest)
	if lag < 0 {
		return 0, nil
	}
	return lag, nil
}

// Compile-time assertions that MemoryStore satisfies both ports.
var (
	_ report.ProjectionStore = (*MemoryStore)(nil)
	_ report.ReportStore     = (*MemoryStore)(nil)
)
