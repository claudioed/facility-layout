//go:build integration

package postgres_test

import (
	"testing"

	"github.com/claudioed/facility-layout/internal/adapters/outbound/postgres"
	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/site"
	"github.com/claudioed/facility-layout/internal/domain/slot"
	"github.com/claudioed/facility-layout/internal/domain/structure"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

func mustGeomPointIT(t *testing.T, x, y, z float64) shared.Point3D {
	t.Helper()
	p, err := shared.NewPoint3D(x, y, z)
	if err != nil {
		t.Fatalf("point: %v", err)
	}
	return p
}

func mustGeomDimensionsIT(t *testing.T, w, d, h float64) shared.Dimensions {
	t.Helper()
	dim, err := shared.NewDimensions(w, d, h)
	if err != nil {
		t.Fatalf("dimensions: %v", err)
	}
	return dim
}

// TestPostgresSlotGeometryRoundTrip proves migration 0003's nullable,
// all-or-nothing geometry columns on location_slots really round-trip
// against a live database: a slot with no geometry set stays NULL across
// every column, and one with geometry set comes back with the exact
// position/dimensions/pick-sequence it was given (ADR-0017).
func TestPostgresSlotGeometryRoundTrip(t *testing.T) {
	ctx, pool := newPool(t)

	sites := postgres.NewSiteRepo(pool)
	zones := postgres.NewZoneRepo(pool)
	aisles := postgres.NewAisleRepo(pool)
	types := postgres.NewLocationTypeRepo(pool)
	slots := postgres.NewSlotRepo(pool)

	s, _ := site.NewSite("WH1", "Fulfilment Centre One")
	if err := sites.Save(ctx, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	z, _ := zone.NewZone("WH1", "STOR", "AMB", shared.Ambient, false)
	if err := zones.Save(ctx, z); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a, _ := aisle.NewAisle("WH1-STOR-AMB", "A07", 7, shared.TwoWay)
	if err := aisles.Save(ctx, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	palletRack := mustLocationType(t, placement.PalletRack, 1200, 2.4)
	if err := types.Save(ctx, palletRack); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := placement.ZoneAttributes{ZoneID: z.ID(), ZoneCode: z.ZoneCode(), TemperatureClass: z.TemperatureClass()}
	code := mustCode(t, "WH1-STOR-AMB-A07-03-02-B")
	built, err := slot.NewLocationSlot(code, palletRack, shared.Capacity{}, slot.FunctionalAttributes{}, attrs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := slots.Save(ctx, built); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("a freshly registered slot has no geometry", func(t *testing.T) {
		found, err := slots.FindByCode(ctx, code)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found.Position().IsZero() || !found.Dimensions().IsZero() || found.PickSequence() != nil {
			t.Fatalf("expected no geometry, got position=%+v dimensions=%+v pickSequence=%v", found.Position(), found.Dimensions(), found.PickSequence())
		}
	})

	t.Run("setting geometry round-trips position, dimensions, and pick sequence", func(t *testing.T) {
		found, err := slots.FindByCode(ctx, code)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		position := mustGeomPointIT(t, 12.5, 7.25, 0)
		dimensions := mustGeomDimensionsIT(t, 1.2, 0.9, 2.4)
		if err := found.SetGeometry(position, dimensions); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := found.SetPickSequence(42); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := slots.Save(ctx, found); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reloaded, err := slots.FindByCode(ctx, code)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reloaded.Position() != position {
			t.Fatalf("expected position %+v, got %+v", position, reloaded.Position())
		}
		if reloaded.Dimensions() != dimensions {
			t.Fatalf("expected dimensions %+v, got %+v", dimensions, reloaded.Dimensions())
		}
		if reloaded.PickSequence() == nil || *reloaded.PickSequence() != 42 {
			t.Fatalf("expected pick sequence 42, got %v", reloaded.PickSequence())
		}
	})
}

// TestPostgresAisleCentrelineRoundTrip proves migration 0003's nullable,
// all-or-nothing centreline columns on aisles round-trip against a live
// database (ADR-0017).
func TestPostgresAisleCentrelineRoundTrip(t *testing.T) {
	ctx, pool := newPool(t)

	sites := postgres.NewSiteRepo(pool)
	zones := postgres.NewZoneRepo(pool)
	aisles := postgres.NewAisleRepo(pool)

	s, _ := site.NewSite("WH1", "Fulfilment Centre One")
	if err := sites.Save(ctx, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	z, _ := zone.NewZone("WH1", "STOR", "AMB", shared.Ambient, false)
	if err := zones.Save(ctx, z); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a, _ := aisle.NewAisle("WH1-STOR-AMB", "A07", 7, shared.TwoWay)
	if err := aisles.Save(ctx, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("a freshly registered aisle has no centreline", func(t *testing.T) {
		found, err := aisles.FindByID(ctx, "WH1-STOR-AMB-A07")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found.Centreline().IsZero() {
			t.Fatalf("expected no centreline, got %+v", found.Centreline())
		}
	})

	t.Run("setting a centreline round-trips both endpoints", func(t *testing.T) {
		found, err := aisles.FindByID(ctx, "WH1-STOR-AMB-A07")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		centreline, err := shared.NewSegment(mustGeomPointIT(t, 0, 0, 0), mustGeomPointIT(t, 24, 0, 0))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := found.SetCentreline(centreline); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := aisles.Save(ctx, found); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reloaded, err := aisles.FindByID(ctx, "WH1-STOR-AMB-A07")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reloaded.Centreline() != centreline {
			t.Fatalf("expected centreline %+v, got %+v", centreline, reloaded.Centreline())
		}
	})
}

// TestPostgresFixedStructureRepoRoundTrip proves the new fixed_structures
// table (migration 0003) round-trips a full FixedStructure aggregate
// against a live database (ADR-0017).
func TestPostgresFixedStructureRepoRoundTrip(t *testing.T) {
	ctx, pool := newPool(t)

	sites := postgres.NewSiteRepo(pool)
	structures := postgres.NewFixedStructureRepo(pool)

	s, _ := site.NewSite("WH1", "Fulfilment Centre One")
	if err := sites.Save(ctx, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	origin := mustGeomPointIT(t, 10, 20, 0)
	size := mustGeomDimensionsIT(t, 1, 1, 3)
	footprint, err := shared.NewRect(origin, size)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wall, err := structure.NewFixedStructure("STR-1", "WH1", structure.Wall, footprint, "North wall")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	column, err := structure.NewFixedStructure("STR-2", "WH1", structure.Column, footprint, "Column B3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range []*structure.FixedStructure{wall, column} {
		if err := structures.Save(ctx, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	found, err := structures.FindByID(ctx, "STR-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil || found.Kind() != structure.Wall || found.Label() != "North wall" || found.Footprint() != footprint {
		t.Fatalf("unexpected structure round-trip %+v", found)
	}

	if missing, _ := structures.FindByID(ctx, "STR-NOPE"); missing != nil {
		t.Fatal("expected (nil, nil) for an unknown structure")
	}

	all, err := structures.ListBySite(ctx, "WH1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 || all[0].ID() != "STR-1" {
		t.Fatalf("expected structures ordered by id, got %+v", all)
	}
}
