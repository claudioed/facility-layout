package analyticsstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/claudioed/facility-layout/internal/adapters/outbound/analyticsstore"
	"github.com/claudioed/facility-layout/internal/analytics/report"
)

func TestMemoryStore_ProjectsAndDedupes(t *testing.T) {
	base := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	ctx := context.Background()
	s := analyticsstore.NewMemoryStore()

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
	}

	must(s.ApplySiteRegistered(ctx, "e1", "WH1", base))
	must(s.ApplyZoneRegistered(ctx, "e2", "WH1", base))
	// duplicate slot event id -> counts once
	must(s.ApplyLocationSlotRegistered(ctx, "dup", "WH1-STOR-AMB", base))
	must(s.ApplyLocationSlotRegistered(ctx, "dup", "WH1-STOR-AMB", base))
	must(s.ApplyFacilityLayoutImported(ctx, "e3", "", 100, 97, 3, base))

	rep, err := s.Query(ctx, report.ReportQuery{
		From:        base.Add(-24 * time.Hour),
		To:          base.Add(24 * time.Hour),
		Granularity: report.GranularityDay,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	tests := []struct {
		name    string
		scope   string
		check   func(report.Row) bool
		wantMsg string
	}{
		{"site row", "WH1", func(r report.Row) bool { return r.SitesRegistered == 1 && r.ZonesRegistered == 1 }, "sites=1 zones=1"},
		{"zone row deduped", "WH1-STOR-AMB", func(r report.Row) bool { return r.SlotsRegistered == 1 }, "slots=1 (deduped)"},
		{"catalog import row", "", func(r report.Row) bool {
			return r.BulkImports == 1 && r.ImportRowsSubmitted == 100 && r.ImportRowsImported == 97 && r.ImportRowsRejected == 3
		}, "import tallies"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found *report.Row
			for i := range rep.Rows {
				if rep.Rows[i].Key.Scope == tt.scope {
					found = &rep.Rows[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("no row for scope %q", tt.scope)
			}
			if !tt.check(*found) {
				t.Errorf("row %+v failed check %s", *found, tt.wantMsg)
			}
		})
	}
}

func TestMemoryStore_FreshnessLag(t *testing.T) {
	ctx := context.Background()

	t.Run("empty store is zero lag", func(t *testing.T) {
		s := analyticsstore.NewMemoryStore()
		lag, err := s.FreshnessLag(ctx)
		if err != nil {
			t.Fatalf("FreshnessLag: %v", err)
		}
		if lag != 0 {
			t.Errorf("lag = %v, want 0", lag)
		}
	})

	t.Run("lag from latest applied event", func(t *testing.T) {
		latest := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
		now := latest.Add(90 * time.Second)
		s := analyticsstore.NewMemoryStore()
		s.Now = func() time.Time { return now }
		if err := s.ApplySiteRegistered(ctx, "e1", "WH1", latest); err != nil {
			t.Fatalf("apply: %v", err)
		}
		lag, err := s.FreshnessLag(ctx)
		if err != nil {
			t.Fatalf("FreshnessLag: %v", err)
		}
		if lag != 90*time.Second {
			t.Errorf("lag = %v, want 90s", lag)
		}
	})
}
