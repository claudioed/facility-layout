//go:build integration

package postgres_test

import (
	"testing"

	"github.com/claudioed/facility-layout/internal/adapters/outbound/postgres"
	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/site"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

// TestPostgresCrossAisleRepoRoundTrip proves migration 0004's cross_aisles
// table really round-trips a CrossAisle against a live database, including
// the symmetric FindByAisles lookup and the decommissioned flag (ADR-0017).
func TestPostgresCrossAisleRepoRoundTrip(t *testing.T) {
	ctx, pool := newPool(t)

	sites := postgres.NewSiteRepo(pool)
	zones := postgres.NewZoneRepo(pool)
	aisles := postgres.NewAisleRepo(pool)
	crossAisles := postgres.NewCrossAisleRepo(pool)

	s, _ := site.NewSite("WH1", "Fulfilment Centre One")
	if err := sites.Save(ctx, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	z, _ := zone.NewZone("WH1", "STOR", "AMB", shared.Ambient, false)
	if err := zones.Save(ctx, z); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a07, _ := aisle.NewAisle("WH1-STOR-AMB", "A07", 7, shared.TwoWay)
	if err := aisles.Save(ctx, a07); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a08, _ := aisle.NewAisle("WH1-STOR-AMB", "A08", 8, shared.TwoWay)
	if err := aisles.Save(ctx, a08); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, err := aisle.NewCrossAisle("WH1-STOR-AMB", "A07", "A08", "02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := crossAisles.Save(ctx, c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("FindByAisles resolves regardless of orientation", func(t *testing.T) {
		found, err := crossAisles.FindByAisles(ctx, "WH1-STOR-AMB", "A08", "A07", "02")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found == nil {
			t.Fatal("expected the cross-aisle to be found from the reverse orientation")
		}
		if !found.IsActive() {
			t.Fatal("expected an active cross-aisle")
		}
	})

	t.Run("ListByZone returns it", func(t *testing.T) {
		list, err := crossAisles.ListByZone(ctx, "WH1-STOR-AMB")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 cross-aisle, got %d", len(list))
		}
	})

	t.Run("decommissioning round-trips", func(t *testing.T) {
		if err := c.Decommission(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := crossAisles.Save(ctx, c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found, err := crossAisles.FindByAisles(ctx, "WH1-STOR-AMB", "A07", "A08", "02")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.IsActive() {
			t.Fatal("expected the persisted cross-aisle to be decommissioned")
		}
	})

	t.Run("FindByAisles returns nil for a connection that was never registered", func(t *testing.T) {
		found, err := crossAisles.FindByAisles(ctx, "WH1-STOR-AMB", "A07", "A08", "99")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil, got %+v", found)
		}
	})
}

// TestPostgresZonePitchRoundTrip proves migration 0004's nullable
// bay_pitch_m/level_pitch_m columns on zones round-trip: a zone that never
// sets an override reads back the domain default, and one with an
// explicit override reads back exactly that value (ADR-0017).
func TestPostgresZonePitchRoundTrip(t *testing.T) {
	ctx, pool := newPool(t)

	sites := postgres.NewSiteRepo(pool)
	zones := postgres.NewZoneRepo(pool)

	s, _ := site.NewSite("WH2", "Fulfilment Centre Two")
	if err := sites.Save(ctx, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("default pitch when never set", func(t *testing.T) {
		z, _ := zone.NewZone("WH2", "STOR", "AMB", shared.Ambient, false)
		if err := zones.Save(ctx, z); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found, err := zones.FindByID(ctx, "WH2-STOR-AMB")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.BayPitchM() != zone.DefaultBayPitchM || found.LevelPitchM() != zone.DefaultLevelPitchM {
			t.Fatalf("expected default pitch, got (%v, %v)", found.BayPitchM(), found.LevelPitchM())
		}
	})

	t.Run("explicit pitch persists", func(t *testing.T) {
		z, _ := zone.NewZone("WH2", "STOR", "FRZ", shared.Frozen, false)
		if err := z.SetPitch(2.0, 3.0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := zones.Save(ctx, z); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found, err := zones.FindByID(ctx, "WH2-STOR-FRZ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.BayPitchM() != 2.0 || found.LevelPitchM() != 3.0 {
			t.Fatalf("expected pitch (2.0, 3.0), got (%v, %v)", found.BayPitchM(), found.LevelPitchM())
		}
	})
}
