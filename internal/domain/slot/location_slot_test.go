package slot_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/slot"
)

func mustCode(t *testing.T, raw string) shared.LocationCode {
	t.Helper()
	code, err := shared.ParseLocationCode(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return code
}

func mustCapacity(t *testing.T, weight, volume float64) shared.Capacity {
	t.Helper()
	capacity, err := shared.NewCapacity(weight, volume)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return capacity
}

func mustLocationType(t *testing.T, name string, weight, volume float64) placement.LocationType {
	t.Helper()
	lt, err := placement.NewLocationType(name, mustCapacity(t, weight, volume))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return lt
}

func mustRule(t *testing.T, id, locationType string, effect placement.Effect, zoneCode string, temperatureClass shared.TemperatureClass) placement.PlacementRule {
	t.Helper()
	predicate, err := placement.NewZonePredicate(zoneCode, temperatureClass, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rule, err := placement.NewPlacementRule(id, locationType, effect, predicate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return rule
}

func TestNewLocationSlot(t *testing.T) {
	ambientAttrs := placement.ZoneAttributes{ZoneID: "WH1-STOR-AMB", ZoneCode: "AMB", TemperatureClass: shared.Ambient}
	palletRack := mustLocationType(t, placement.PalletRack, 1200, 2.4)

	t.Run("uses the location type's default capacity envelope", func(t *testing.T) {
		s, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), palletRack, shared.Capacity{}, ambientAttrs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Capacity().MaxWeightKg() != 1200 || s.Capacity().MaxVolumeM3() != 2.4 {
			t.Fatalf("expected the type default envelope, got %v", s.Capacity())
		}
		if s.Code().String() != "WH1-STOR-AMB-A07-03-02-B" || s.LocationType() != placement.PalletRack {
			t.Fatalf("unexpected slot %q/%q", s.Code(), s.LocationType())
		}
		if !s.IsActive() || s.Status() != shared.Active {
			t.Fatalf("a newly registered slot must be Active, got %q", s.Status())
		}
	})

	t.Run("an explicit override beats the type default", func(t *testing.T) {
		s, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), palletRack, mustCapacity(t, 500, 1.1), ambientAttrs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Capacity().MaxWeightKg() != 500 || s.Capacity().MaxVolumeM3() != 1.1 {
			t.Fatalf("expected the override envelope, got %v", s.Capacity())
		}
	})

	t.Run("a zero location code is rejected", func(t *testing.T) {
		_, err := slot.NewLocationSlot(shared.LocationCode{}, palletRack, shared.Capacity{}, ambientAttrs, nil)
		if !errors.Is(err, slot.ErrMissingLocationCode) {
			t.Fatalf("expected ErrMissingLocationCode, got %v", err)
		}
	})

	t.Run("a zero location type is rejected", func(t *testing.T) {
		_, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), placement.LocationType{}, shared.Capacity{}, ambientAttrs, nil)
		if !errors.Is(err, slot.ErrMissingLocationType) {
			t.Fatalf("expected ErrMissingLocationType, got %v", err)
		}
	})

	t.Run("zone attributes for a different zone are rejected", func(t *testing.T) {
		_, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-FRZ-A07-03-02-B"), palletRack, shared.Capacity{}, ambientAttrs, nil)
		if !errors.Is(err, slot.ErrZoneMismatch) {
			t.Fatalf("expected ErrZoneMismatch, got %v", err)
		}
	})

	t.Run("a location type with no default capacity and no override is rejected", func(t *testing.T) {
		bare := placement.RehydrateLocationType(placement.Staging, shared.Capacity{})
		_, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), bare, shared.Capacity{}, ambientAttrs, nil)
		if !errors.Is(err, shared.ErrInvalidMaxWeight) {
			t.Fatalf("expected ErrInvalidMaxWeight, got %v", err)
		}
	})
}

func TestNewLocationSlotEnforcesPlacementRules(t *testing.T) {
	hazAttrs := placement.ZoneAttributes{ZoneID: "WH1-STOR-HAZ", ZoneCode: "HAZ", TemperatureClass: shared.Ambient, Hazmat: true}
	frozenAttrs := placement.ZoneAttributes{ZoneID: "WH1-STOR-FRZ", ZoneCode: "FRZ", TemperatureClass: shared.Frozen}

	palletRack := mustLocationType(t, placement.PalletRack, 1200, 2.4)
	shelf := mustLocationType(t, placement.Shelf, 60, 0.4)

	rules := placement.RuleSet{
		mustRule(t, "RULE-HAZ-ONLY-RACK", placement.PalletRack, placement.Allow, "HAZ", ""),
		mustRule(t, "RULE-FRZ-NO-SHELF", placement.Shelf, placement.Deny, "", shared.Frozen),
	}

	t.Run("a legal placement is accepted", func(t *testing.T) {
		if _, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-HAZ-A01-01-01-A"), palletRack, shared.Capacity{}, hazAttrs, rules); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("a type outside the zone's allow-list is rejected naming the rule", func(t *testing.T) {
		_, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-HAZ-A01-01-01-A"), shelf, shared.Capacity{}, hazAttrs, rules)
		if !errors.Is(err, placement.ErrPlacementRuleViolated) {
			t.Fatalf("expected ErrPlacementRuleViolated, got %v", err)
		}
		if !strings.Contains(err.Error(), "RULE-HAZ-ONLY-RACK") {
			t.Fatalf("expected the violated rule to be named, got %q", err.Error())
		}
	})

	t.Run("a denied type in a frozen zone is rejected naming the rule", func(t *testing.T) {
		_, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-FRZ-A02-01-01-A"), shelf, shared.Capacity{}, frozenAttrs, rules)
		if !errors.Is(err, placement.ErrPlacementRuleViolated) {
			t.Fatalf("expected ErrPlacementRuleViolated, got %v", err)
		}
		if !strings.Contains(err.Error(), "RULE-FRZ-NO-SHELF") {
			t.Fatalf("expected the violated rule to be named, got %q", err.Error())
		}
	})
}

func TestLocationSlotDecommissionIsOneWay(t *testing.T) {
	attrs := placement.ZoneAttributes{ZoneID: "WH1-STOR-AMB", ZoneCode: "AMB", TemperatureClass: shared.Ambient}
	s, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), mustLocationType(t, placement.PalletRack, 1200, 2.4), shared.Capacity{}, attrs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := s.Decommission(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.IsActive() || s.Status() != shared.Decommissioned {
		t.Fatalf("expected a Decommissioned slot, got %q", s.Status())
	}
	if err := s.Decommission(); !errors.Is(err, slot.ErrAlreadyDecommissioned) {
		t.Fatalf("expected ErrAlreadyDecommissioned, got %v", err)
	}
}

func TestRehydrateLocationSlot(t *testing.T) {
	code := mustCode(t, "WH1-RCV-AMB-D01-01-01-A")
	s := slot.RehydrateLocationSlot(code, placement.Staging, mustCapacity(t, 900, 3), shared.UnderMaintenance)

	if s.Code() != code || s.LocationType() != placement.Staging {
		t.Fatalf("unexpected rehydrated slot %q/%q", s.Code(), s.LocationType())
	}
	if s.IsActive() {
		t.Fatal("an UnderMaintenance slot must not report as Active")
	}
	if err := s.Decommission(); err != nil {
		t.Fatalf("an UnderMaintenance slot must still be decommissionable: %v", err)
	}
	if s.Status() != shared.Decommissioned {
		t.Fatalf("expected Decommissioned, got %q", s.Status())
	}
}
