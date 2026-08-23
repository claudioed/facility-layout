package usecases_test

import (
	"strings"
	"testing"

	"github.com/claudioed/facility-layout/internal/application/usecases"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/slot"
)

func TestRegisterLocationSlot(t *testing.T) {
	t.Run("registers a slot whose whole chain of custody resolves", func(t *testing.T) {
		h := newHarness(t)
		h.seedAmbientAisle()

		s, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), placement.PalletRack, shared.Capacity{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Code().String() != "WH1-STOR-AMB-A07-03-02-B" || !s.IsActive() {
			t.Fatalf("unexpected slot %+v", s)
		}
		if s.Capacity().MaxWeightKg() != 1200 {
			t.Fatalf("expected the location type's default envelope, got %v", s.Capacity())
		}
		h.assertPublished("LocationSlotRegistered")
	})

	t.Run("honours a capacity override", func(t *testing.T) {
		h := newHarness(t)
		h.seedAmbientAisle()
		s, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), placement.PalletRack, mustCapacity(t, 400, 0.9))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Capacity().MaxWeightKg() != 400 || s.Capacity().MaxVolumeM3() != 0.9 {
			t.Fatalf("expected the override envelope, got %v", s.Capacity())
		}
	})

	t.Run("rejects a duplicate location code", func(t *testing.T) {
		h := newHarness(t)
		h.seedAmbientAisle()
		h.mustRegisterSlot("WH1-STOR-AMB-A07-03-02-B", placement.PalletRack)
		_, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), placement.PalletRack, shared.Capacity{})
		assertErrorIs(t, err, usecases.ErrDuplicateLocationCode)
	})

	t.Run("rejects re-registering a decommissioned code", func(t *testing.T) {
		h := newHarness(t)
		h.seedAmbientAisle()
		h.mustRegisterSlot("WH1-STOR-AMB-A07-03-02-B", placement.PalletRack)
		if err := h.decommissionSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), placement.PalletRack, shared.Capacity{})
		assertErrorIs(t, err, usecases.ErrDuplicateLocationCode)
	})

	t.Run("rejects an unknown location type", func(t *testing.T) {
		h := newHarness(t)
		h.seedAmbientAisle()
		_, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), "Hovercraft", shared.Capacity{})
		assertErrorIs(t, err, usecases.ErrLocationTypeNotFound)
	})
}

func TestRegisterLocationSlotChainOfCustody(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, h *harness)
		code    string
		wantErr error
	}{
		{
			name:    "unknown site",
			setup:   func(_ *testing.T, h *harness) { h.seedAmbientAisle() },
			code:    "WH9-STOR-AMB-A07-03-02-B",
			wantErr: usecases.ErrSiteNotFound,
		},
		{
			name: "decommissioned site",
			setup: func(t *testing.T, h *harness) {
				h.seedAmbientAisle()
				decommissionSite(t, h, "WH1")
			},
			code:    "WH1-STOR-AMB-A07-03-02-B",
			wantErr: usecases.ErrSiteNotActive,
		},
		{
			name:    "unknown zone",
			setup:   func(_ *testing.T, h *harness) { h.seedAmbientAisle() },
			code:    "WH1-STOR-FRZ-A07-03-02-B",
			wantErr: usecases.ErrZoneNotFound,
		},
		{
			name: "decommissioned zone",
			setup: func(t *testing.T, h *harness) {
				h.seedAmbientAisle()
				decommissionZone(t, h, "WH1-STOR-AMB")
			},
			code:    "WH1-STOR-AMB-A07-03-02-B",
			wantErr: usecases.ErrZoneNotActive,
		},
		{
			name:    "unknown aisle",
			setup:   func(_ *testing.T, h *harness) { h.seedAmbientAisle() },
			code:    "WH1-STOR-AMB-A99-03-02-B",
			wantErr: usecases.ErrAisleNotFound,
		},
		{
			name: "decommissioned aisle",
			setup: func(t *testing.T, h *harness) {
				h.seedAmbientAisle()
				decommissionAisle(t, h, "WH1-STOR-AMB-A07")
			},
			code:    "WH1-STOR-AMB-A07-03-02-B",
			wantErr: usecases.ErrAisleNotActive,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			tc.setup(t, h)

			_, err := h.registerSlot.Execute(h.ctx(), mustCode(t, tc.code), placement.PalletRack, shared.Capacity{})
			assertErrorIs(t, err, tc.wantErr)
			h.assertNotPublished("LocationSlotRegistered")
		})
	}
}

func TestRegisterLocationSlotEnforcesPlacementRules(t *testing.T) {
	newHazmatHarness := func(t *testing.T) *harness {
		t.Helper()
		h := newHarness(t)
		h.mustRegisterSite("WH1", "Fulfilment Centre One")
		h.mustRegisterZone("WH1", "STOR", "HAZ", shared.Ambient, true)
		h.mustRegisterAisle("WH1-STOR-HAZ", "A01", 1, shared.OneWay)
		h.mustRegisterZone("WH1", "STOR", "FRZ", shared.Frozen, false)
		h.mustRegisterAisle("WH1-STOR-FRZ", "A02", 2, shared.TwoWay)
		h.mustRegisterLocationType(placement.PalletRack, 1200, 2.4)
		h.mustRegisterLocationType(placement.Shelf, 60, 0.4)
		mustDefineRule(t, h, "RULE-HAZ-ONLY-RACK", placement.PalletRack, placement.Allow, mustPredicate(t, "HAZ", "", nil))
		mustDefineRule(t, h, "RULE-FRZ-NO-SHELF", placement.Shelf, placement.Deny, mustPredicate(t, "", shared.Frozen, nil))
		return h
	}

	t.Run("a placement satisfying every rule is accepted", func(t *testing.T) {
		h := newHazmatHarness(t)
		if _, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-HAZ-A01-01-01-A"), placement.PalletRack, shared.Capacity{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("a type outside the zone's allow-list is rejected naming the rule", func(t *testing.T) {
		h := newHazmatHarness(t)
		_, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-HAZ-A01-01-01-A"), placement.Shelf, shared.Capacity{})
		assertErrorIs(t, err, placement.ErrPlacementRuleViolated)
		if !strings.Contains(err.Error(), "RULE-HAZ-ONLY-RACK") {
			t.Fatalf("expected the violated rule to be named, got %q", err.Error())
		}
		h.assertNotPublished("LocationSlotRegistered")
	})

	t.Run("a denied type in a frozen zone is rejected naming the rule", func(t *testing.T) {
		h := newHazmatHarness(t)
		_, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-FRZ-A02-01-01-A"), placement.Shelf, shared.Capacity{})
		assertErrorIs(t, err, placement.ErrPlacementRuleViolated)
		if !strings.Contains(err.Error(), "RULE-FRZ-NO-SHELF") {
			t.Fatalf("expected the violated rule to be named, got %q", err.Error())
		}
	})
}

func TestGetLocationSlot(t *testing.T) {
	h := newHarness(t)
	h.seedAmbientAisle()
	h.mustRegisterSlot("WH1-STOR-AMB-A07-03-02-B", placement.PalletRack)

	got, err := h.getSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.LocationType() != placement.PalletRack {
		t.Fatalf("unexpected slot %+v", got)
	}

	_, err = h.getSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-99-99-Z"))
	assertErrorIs(t, err, usecases.ErrLocationSlotNotFound)
}

func TestDecommissionLocationSlot(t *testing.T) {
	t.Run("decommissions an active slot", func(t *testing.T) {
		h := newHarness(t)
		h.seedAmbientAisle()
		h.mustRegisterSlot("WH1-STOR-AMB-A07-03-02-B", placement.PalletRack)

		if err := h.decommissionSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		h.assertPublished("LocationSlotDecommissioned")

		stored, err := h.slots.FindByCode(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stored.IsActive() {
			t.Fatal("expected the persisted slot to be Decommissioned")
		}
	})

	t.Run("rejects an unknown slot", func(t *testing.T) {
		h := newHarness(t)
		err := h.decommissionSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"))
		assertErrorIs(t, err, usecases.ErrLocationSlotNotFound)
	})

	t.Run("decommission is one-way", func(t *testing.T) {
		h := newHarness(t)
		h.seedAmbientAisle()
		h.mustRegisterSlot("WH1-STOR-AMB-A07-03-02-B", placement.PalletRack)
		code := mustCode(t, "WH1-STOR-AMB-A07-03-02-B")
		if err := h.decommissionSlot.Execute(h.ctx(), code); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err := h.decommissionSlot.Execute(h.ctx(), code)
		assertErrorIs(t, err, slot.ErrAlreadyDecommissioned)
	})
}
