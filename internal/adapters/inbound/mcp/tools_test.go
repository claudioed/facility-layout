package mcp

import (
	"context"
	"testing"

	"github.com/claudioed/facility-layout/internal/adapters/outbound/events"
	"github.com/claudioed/facility-layout/internal/adapters/outbound/memory"
	"github.com/claudioed/facility-layout/internal/application/usecases"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// harness wires the read use cases the MCP adapter needs over in-memory
// repos, seeding structure through the REAL write use cases so the read
// models under test are built from genuinely-registered aggregates.
type harness struct {
	t *testing.T

	sites         *memory.SiteRepo
	zones         *memory.ZoneRepo
	aisles        *memory.AisleRepo
	slots         *memory.SlotRepo
	locationTypes *memory.LocationTypeRepo
	rules         *memory.PlacementRuleRepo
	clock         *memory.FixedClock

	registerSite         *usecases.RegisterSite
	registerZone         *usecases.RegisterZone
	registerAisle        *usecases.RegisterAisle
	registerLocationType *usecases.RegisterLocationType
	registerSlot         *usecases.RegisterLocationSlot

	deps Deps
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	sites := memory.NewSiteRepo()
	zones := memory.NewZoneRepo()
	aisles := memory.NewAisleRepo()
	slots := memory.NewSlotRepo()
	locationTypes := memory.NewLocationTypeRepo()
	rules := memory.NewPlacementRuleRepo()
	publisher := events.NewBufferedPublisher()
	clock := memory.NewFixedClock(base)

	h := &harness{
		t:             t,
		sites:         sites,
		zones:         zones,
		aisles:        aisles,
		slots:         slots,
		locationTypes: locationTypes,
		rules:         rules,
		clock:         clock,

		registerSite:         &usecases.RegisterSite{Sites: sites, Events: publisher, Clock: clock},
		registerZone:         &usecases.RegisterZone{Sites: sites, Zones: zones, Events: publisher, Clock: clock},
		registerAisle:        &usecases.RegisterAisle{Zones: zones, Aisles: aisles, Events: publisher, Clock: clock},
		registerLocationType: &usecases.RegisterLocationType{LocationTypes: locationTypes, Events: publisher, Clock: clock},
		registerSlot: &usecases.RegisterLocationSlot{
			Sites: sites, Zones: zones, Aisles: aisles, Slots: slots,
			LocationTypes: locationTypes, Rules: rules, Events: publisher, Clock: clock,
		},
	}
	h.deps = Deps{
		GetSiteLayout: &usecases.GetSiteLayout{Sites: sites, Zones: zones, Aisles: aisles, Slots: slots},
		GetZoneGrid:   &usecases.GetZoneGrid{Zones: zones, Aisles: aisles, Slots: slots},
		ListSites:     &usecases.ListSites{Sites: sites},
	}
	return h
}

func (h *harness) ctx() context.Context { return context.Background() }

// seedDrawableSite builds a small but realistic WH1: two zones, three aisles
// registered out of walk order, and a handful of slots — mirrors the
// application-layer read-model test fixture.
func (h *harness) seedDrawableSite() {
	h.t.Helper()
	h.mustRegisterSite("WH1", "Fulfilment Centre One")
	h.mustRegisterLocationType(placement.PalletRack, 1200, 2.4)
	h.mustRegisterLocationType(placement.Shelf, 60, 0.4)

	h.mustRegisterZone("WH1", "STOR", "AMB", shared.Ambient, false)
	h.mustRegisterZone("WH1", "RCV", "AMB", shared.Ambient, false)

	// Registered deliberately out of walk order: A09 first, then A07.
	h.mustRegisterAisle("WH1-STOR-AMB", "A09", 9, shared.OneWay)
	h.mustRegisterAisle("WH1-STOR-AMB", "A07", 7, shared.TwoWay)
	h.mustRegisterAisle("WH1-RCV-AMB", "D01", 1, shared.TwoWay)

	h.mustRegisterSlot("WH1-STOR-AMB-A07-03-02-B", placement.PalletRack)
	h.mustRegisterSlot("WH1-STOR-AMB-A07-03-02-A", placement.PalletRack)
	h.mustRegisterSlot("WH1-STOR-AMB-A07-03-01-A", placement.PalletRack)
	h.mustRegisterSlot("WH1-STOR-AMB-A09-01-01-A", placement.Shelf)
	h.mustRegisterSlot("WH1-RCV-AMB-D01-01-01-A", placement.Shelf)
}

func (h *harness) mustRegisterSite(code, name string) {
	h.t.Helper()
	if _, err := h.registerSite.Execute(h.ctx(), code, name); err != nil {
		h.t.Fatalf("seeding site %q: %v", code, err)
	}
}

func (h *harness) mustRegisterZone(siteCode, areaCode, zoneCode string, tc shared.TemperatureClass, hazmat bool) {
	h.t.Helper()
	if _, err := h.registerZone.Execute(h.ctx(), siteCode, areaCode, zoneCode, tc, hazmat); err != nil {
		h.t.Fatalf("seeding zone %s-%s-%s: %v", siteCode, areaCode, zoneCode, err)
	}
}

func (h *harness) mustRegisterAisle(zoneID, aisleCode string, sequenceHint int, direction shared.Direction) {
	h.t.Helper()
	if _, err := h.registerAisle.Execute(h.ctx(), zoneID, aisleCode, sequenceHint, direction); err != nil {
		h.t.Fatalf("seeding aisle %s-%s: %v", zoneID, aisleCode, err)
	}
}

func (h *harness) mustRegisterLocationType(name string, weight, volume float64) {
	h.t.Helper()
	capacity, err := shared.NewCapacity(weight, volume)
	if err != nil {
		h.t.Fatalf("capacity: %v", err)
	}
	if _, err := h.registerLocationType.Execute(h.ctx(), name, capacity); err != nil {
		h.t.Fatalf("seeding location type %q: %v", name, err)
	}
}

func (h *harness) mustRegisterSlot(raw, locationType string) {
	h.t.Helper()
	code, err := shared.ParseLocationCode(raw)
	if err != nil {
		h.t.Fatalf("parse code %q: %v", raw, err)
	}
	if _, err := h.registerSlot.Execute(h.ctx(), code, locationType, shared.Capacity{}); err != nil {
		h.t.Fatalf("seeding slot %q: %v", raw, err)
	}
}

func TestListSites(t *testing.T) {
	h := newHarness(t)

	// Empty map -> empty, non-nil list.
	empty, err := h.deps.listSites(h.ctx(), listSitesInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(empty.Sites) != 0 {
		t.Fatalf("expected no sites, got %+v", empty.Sites)
	}

	h.mustRegisterSite("WH1", "Fulfilment Centre One")
	h.mustRegisterSite("WH2", "Fulfilment Centre Two")

	out, err := h.deps.listSites(h.ctx(), listSitesInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Sites) != 2 {
		t.Fatalf("expected 2 sites, got %d", len(out.Sites))
	}
	// Repo returns them ordered by code.
	if out.Sites[0].Code != "WH1" || out.Sites[0].Name != "Fulfilment Centre One" {
		t.Fatalf("unexpected first site %+v", out.Sites[0])
	}
	if out.Sites[1].Code != "WH2" {
		t.Fatalf("unexpected second site %+v", out.Sites[1])
	}
}

func TestGetSiteLayout(t *testing.T) {
	tests := []struct {
		name     string
		siteCode string
		wantErr  bool
		assert   func(t *testing.T, out siteLayoutDTO)
	}{
		{
			name:     "empty siteCode rejected",
			siteCode: "",
			wantErr:  true,
		},
		{
			name:     "unknown site rejected",
			siteCode: "NOPE",
			wantErr:  true,
		},
		{
			name:     "full nested drawable structure",
			siteCode: "WH1",
			assert: func(t *testing.T, out siteLayoutDTO) {
				if out.Site.Code != "WH1" || out.Site.Name != "Fulfilment Centre One" {
					t.Fatalf("unexpected site header %+v", out.Site)
				}
				if len(out.Zones) != 2 {
					t.Fatalf("expected 2 zones, got %d", len(out.Zones))
				}
				// Zones ordered by id: RCV before STOR.
				if out.Zones[0].ZoneID != "WH1-RCV-AMB" || out.Zones[1].ZoneID != "WH1-STOR-AMB" {
					t.Fatalf("zones out of order: %s then %s", out.Zones[0].ZoneID, out.Zones[1].ZoneID)
				}
				storage := out.Zones[1]
				if storage.TemperatureClass != string(shared.Ambient) || storage.Hazmat {
					t.Fatalf("unexpected zone attrs %+v", storage)
				}
				if len(storage.Aisles) != 2 {
					t.Fatalf("expected 2 aisles in STOR, got %d", len(storage.Aisles))
				}
				// Walk order (sequenceHint), not registration order: A07 (7) before A09 (9).
				if storage.Aisles[0].AisleCode != "A07" || storage.Aisles[1].AisleCode != "A09" {
					t.Fatalf("aisles not in walk order: %s then %s", storage.Aisles[0].AisleCode, storage.Aisles[1].AisleCode)
				}
				a07 := storage.Aisles[0]
				if a07.SequenceHint != 7 || a07.Direction != string(shared.TwoWay) {
					t.Fatalf("unexpected aisle meta %+v", a07)
				}
				want := []string{"WH1-STOR-AMB-A07-03-01-A", "WH1-STOR-AMB-A07-03-02-A", "WH1-STOR-AMB-A07-03-02-B"}
				if len(a07.SlotCodes) != len(want) {
					t.Fatalf("expected %d slot codes, got %v", len(want), a07.SlotCodes)
				}
				for i, code := range want {
					if a07.SlotCodes[i] != code {
						t.Fatalf("slot %d = %q, want %q", i, a07.SlotCodes[i], code)
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.seedDrawableSite()
			out, err := h.deps.getSiteLayout(h.ctx(), siteLayoutInput{SiteCode: tc.siteCode})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.assert(t, out)
		})
	}
}

func TestGetZoneGrid(t *testing.T) {
	tests := []struct {
		name    string
		zoneID  string
		wantErr bool
		assert  func(t *testing.T, out zoneGridDTO)
	}{
		{"empty zoneId rejected", "", true, nil},
		{"unknown zone rejected", "WH1-NOPE-XXX", true, nil},
		{
			name:   "grid for the storage zone",
			zoneID: "WH1-STOR-AMB",
			assert: func(t *testing.T, out zoneGridDTO) {
				if out.ZoneID != "WH1-STOR-AMB" {
					t.Fatalf("unexpected zone id %q", out.ZoneID)
				}
				if len(out.Columns) == 0 {
					t.Fatal("expected at least one column")
				}
				// Columns carry each aisle's walk-order hint.
				for _, c := range out.Columns {
					if c.AisleCode == "A07" && c.SequenceHint != 7 {
						t.Fatalf("A07 column has wrong sequence hint %d", c.SequenceHint)
					}
				}
				if len(out.Levels) == 0 || len(out.Rows) == 0 {
					t.Fatal("expected levels and rows")
				}
				// Every row's cells are index-aligned with columns.
				for _, r := range out.Rows {
					if len(r.Cells) != len(out.Columns) {
						t.Fatalf("row %q has %d cells, want %d", r.Level, len(r.Cells), len(out.Columns))
					}
				}
				// At least one cell must carry a known slot code.
				found := false
				for _, r := range out.Rows {
					for _, cell := range r.Cells {
						for _, code := range cell.SlotCodes {
							if code == "WH1-STOR-AMB-A07-03-02-B" {
								found = true
							}
						}
					}
				}
				if !found {
					t.Fatal("expected slot WH1-STOR-AMB-A07-03-02-B somewhere in the grid")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.seedDrawableSite()
			out, err := h.deps.getZoneGrid(h.ctx(), zoneGridInput{ZoneID: tc.zoneID})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.assert(t, out)
		})
	}
}
