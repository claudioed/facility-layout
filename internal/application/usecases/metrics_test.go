package usecases_test

import (
	"testing"

	"github.com/claudioed/facility-layout/internal/application/usecases"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// The business metric this service publishes is "slot registrations, by
// outcome". These tests pin the outcome each kind of attempt is counted
// under: the counter is only useful if a placement-rule rejection is
// distinguishable from an ordinary refusal.
func TestRegisterLocationSlotRecordsOutcome(t *testing.T) {
	tests := []struct {
		name string
		// arrange prepares the harness and returns the code to register.
		arrange func(h *harness) shared.LocationCode
		want    string
	}{
		{
			name: "an accepted registration",
			arrange: func(h *harness) shared.LocationCode {
				h.seedAmbientAisle()
				return mustCode(h.t, "WH1-STOR-AMB-A07-03-02-B")
			},
			want: usecases.OutcomeAccepted,
		},
		{
			name: "a registration a placement rule forbids",
			arrange: func(h *harness) shared.LocationCode {
				h.seedAmbientAisle()
				h.mustRegisterLocationType(placement.ToteWall, 60, 0.3)
				denyAll := mustPredicate(h.t, "AMB", "", nil)
				if _, err := h.definePlacementRule.Execute(h.ctx(), "no-totes-in-ambient", placement.ToteWall, placement.Deny, denyAll); err != nil {
					h.t.Fatalf("seeding rule: %v", err)
				}
				return mustCode(h.t, "WH1-STOR-AMB-A07-03-02-C")
			},
			want: usecases.OutcomeRejectedByPlacementRule,
		},
		{
			name: "a registration whose chain of custody does not resolve",
			arrange: func(h *harness) shared.LocationCode {
				h.seedAmbientAisle()
				return mustCode(h.t, "WH9-STOR-AMB-A07-03-02-B")
			},
			want: usecases.OutcomeRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			code := tt.arrange(h)
			h.metrics.outcomes = nil

			locationType := placement.PalletRack
			if tt.want == usecases.OutcomeRejectedByPlacementRule {
				locationType = placement.ToteWall
			}
			_, _ = h.registerSlot.Execute(h.ctx(), code, locationType, shared.Capacity{})

			if len(h.metrics.outcomes) != 1 {
				t.Fatalf("expected exactly one recorded outcome, got %v", h.metrics.outcomes)
			}
			if h.metrics.outcomes[0] != tt.want {
				t.Fatalf("expected outcome %q, got %q", tt.want, h.metrics.outcomes[0])
			}
		})
	}
}

func TestImportFacilityLayoutRecordsOneOutcomePerRow(t *testing.T) {
	h := newHarness(t)
	h.mustRegisterLocationType(placement.PalletRack, 1200, 2.4)
	h.metrics.outcomes = nil

	rows := []usecases.ImportRow{
		{SiteCode: "WH1", AreaCode: "STOR", ZoneCode: "AMB", TemperatureClass: shared.Ambient,
			AisleCode: "A07", Bay: "03", Level: "02", Position: "B", LocationType: placement.PalletRack},
		{SiteCode: "WH1", AreaCode: "STOR", ZoneCode: "AMB", TemperatureClass: shared.Ambient,
			AisleCode: "A07", Bay: "03", Level: "02", Position: "C", LocationType: "NoSuchType"},
	}
	if _, err := h.importLayout.Execute(h.ctx(), rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{usecases.OutcomeAccepted, usecases.OutcomeRejected}
	if len(h.metrics.outcomes) != len(want) {
		t.Fatalf("expected %d recorded outcomes, got %v", len(want), h.metrics.outcomes)
	}
	for i, outcome := range want {
		if h.metrics.outcomes[i] != outcome {
			t.Fatalf("row %d: expected outcome %q, got %q", i, outcome, h.metrics.outcomes[i])
		}
	}
}

// A use case with no recorder wired must behave identically — telemetry is
// never allowed to be load-bearing.
func TestRegisterLocationSlotWithoutMetricsRecorder(t *testing.T) {
	h := newHarness(t)
	h.seedAmbientAisle()
	h.registerSlot.Metrics = nil

	s, err := h.registerSlot.Execute(h.ctx(), mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), placement.PalletRack, shared.Capacity{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Code().String() != "WH1-STOR-AMB-A07-03-02-B" {
		t.Fatalf("unexpected slot %+v", s)
	}
	if len(h.metrics.outcomes) != 0 {
		t.Fatalf("expected nothing recorded, got %v", h.metrics.outcomes)
	}
}
