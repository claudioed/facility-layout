package zone_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

func TestNewZone(t *testing.T) {
	tests := []struct {
		name             string
		siteCode         string
		areaCode         string
		zoneCode         string
		temperatureClass shared.TemperatureClass
		hazmat           bool
		wantID           string
		wantErr          error
	}{
		{name: "ambient storage zone", siteCode: "WH1", areaCode: "STOR", zoneCode: "AMB", temperatureClass: shared.Ambient, wantID: "WH1-STOR-AMB"},
		{name: "hazmat zone", siteCode: "WH1", areaCode: "STOR", zoneCode: "HAZ", temperatureClass: shared.Ambient, hazmat: true, wantID: "WH1-STOR-HAZ"},
		{name: "frozen zone", siteCode: "WH1", areaCode: "STOR", zoneCode: "FRZ", temperatureClass: shared.Frozen, wantID: "WH1-STOR-FRZ"},
		{name: "missing site scope", siteCode: "", areaCode: "STOR", zoneCode: "AMB", temperatureClass: shared.Ambient, wantErr: zone.ErrEmptySiteCode},
		{name: "empty area", siteCode: "WH1", areaCode: "", zoneCode: "AMB", temperatureClass: shared.Ambient, wantErr: zone.ErrEmptyAreaCode},
		{name: "empty zone", siteCode: "WH1", areaCode: "STOR", zoneCode: "", temperatureClass: shared.Ambient, wantErr: zone.ErrEmptyZoneCode},
		{name: "lowercase area", siteCode: "WH1", areaCode: "stor", zoneCode: "AMB", temperatureClass: shared.Ambient, wantErr: zone.ErrInvalidCode},
		{name: "hyphen in zone code", siteCode: "WH1", areaCode: "STOR", zoneCode: "AM-B", temperatureClass: shared.Ambient, wantErr: zone.ErrInvalidCode},
		{name: "unknown temperature class", siteCode: "WH1", areaCode: "STOR", zoneCode: "AMB", temperatureClass: "Tepid", wantErr: shared.ErrUnknownTemperatureClass},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			z, err := zone.NewZone(tc.siteCode, tc.areaCode, tc.zoneCode, tc.temperatureClass, tc.hazmat)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if z.ID() != tc.wantID {
				t.Fatalf("expected zone id %q, got %q", tc.wantID, z.ID())
			}
			if z.SiteCode() != tc.siteCode || z.AreaCode() != tc.areaCode || z.ZoneCode() != tc.zoneCode {
				t.Fatalf("unexpected scoping: %s/%s/%s", z.SiteCode(), z.AreaCode(), z.ZoneCode())
			}
			if z.TemperatureClass() != tc.temperatureClass || z.Hazmat() != tc.hazmat {
				t.Fatalf("unexpected behaviour: %q hazmat=%t", z.TemperatureClass(), z.Hazmat())
			}
			if !z.IsActive() || z.Status() != shared.Active {
				t.Fatalf("a newly registered zone must be Active, got %q", z.Status())
			}
		})
	}
}

func TestZoneDecommissionIsOneWay(t *testing.T) {
	z, err := zone.NewZone("WH1", "STOR", "AMB", shared.Ambient, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := z.Decommission(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if z.IsActive() {
		t.Fatal("expected the zone to stop being Active")
	}
	if err := z.Decommission(); !errors.Is(err, zone.ErrAlreadyDecommissioned) {
		t.Fatalf("expected ErrAlreadyDecommissioned, got %v", err)
	}
}

func TestRehydrateZonePreservesPersistedState(t *testing.T) {
	z := zone.RehydrateZone("WH1", "STOR", "FRZ", shared.Frozen, false, shared.Decommissioned)
	if z.ID() != "WH1-STOR-FRZ" {
		t.Fatalf("unexpected id %q", z.ID())
	}
	if z.IsActive() {
		t.Fatal("a rehydrated Decommissioned zone must not be Active")
	}
	if z.TemperatureClass() != shared.Frozen {
		t.Fatalf("unexpected temperature class %q", z.TemperatureClass())
	}
}
