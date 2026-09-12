package slot_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/slot"
)

func mustNewActiveSlot(t *testing.T) *slot.LocationSlot {
	t.Helper()
	attrs := placement.ZoneAttributes{ZoneID: "WH1-STOR-AMB", ZoneCode: "AMB", TemperatureClass: shared.Ambient}
	s, err := slot.NewLocationSlot(mustCode(t, "WH1-STOR-AMB-A07-03-02-B"), mustLocationType(t, placement.PalletRack, 1200, 2.4), shared.Capacity{}, slot.FunctionalAttributes{}, attrs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return s
}

func TestLocationSlotSetGeometry(t *testing.T) {
	position, err := shared.NewPoint3D(1, 2, 0)
	if err != nil {
		t.Fatalf("position: %v", err)
	}
	dimensions, err := shared.NewDimensions(1.2, 0.9, 2)
	if err != nil {
		t.Fatalf("dimensions: %v", err)
	}

	t.Run("sets position and dimensions on an active slot", func(t *testing.T) {
		s := mustNewActiveSlot(t)
		if !s.Position().IsZero() || !s.Dimensions().IsZero() {
			t.Fatal("expected no geometry before SetGeometry")
		}
		if err := s.SetGeometry(position, dimensions); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Position() != position || s.Dimensions() != dimensions {
			t.Fatalf("unexpected geometry %+v/%+v", s.Position(), s.Dimensions())
		}
	})

	t.Run("rejects a zero position", func(t *testing.T) {
		s := mustNewActiveSlot(t)
		if err := s.SetGeometry(shared.Point3D{}, dimensions); !errors.Is(err, shared.ErrInvalidZ) {
			t.Fatalf("expected ErrInvalidZ, got %v", err)
		}
	})

	t.Run("rejects zero dimensions", func(t *testing.T) {
		s := mustNewActiveSlot(t)
		if err := s.SetGeometry(position, shared.Dimensions{}); !errors.Is(err, shared.ErrInvalidDimensions) {
			t.Fatalf("expected ErrInvalidDimensions, got %v", err)
		}
	})

	t.Run("rejects setting geometry on a decommissioned slot", func(t *testing.T) {
		s := mustNewActiveSlot(t)
		if err := s.Decommission(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := s.SetGeometry(position, dimensions); !errors.Is(err, slot.ErrSlotDecommissioned) {
			t.Fatalf("expected ErrSlotDecommissioned, got %v", err)
		}
	})
}

func TestLocationSlotSetPickSequence(t *testing.T) {
	t.Run("sets an explicit override", func(t *testing.T) {
		s := mustNewActiveSlot(t)
		if s.PickSequence() != nil {
			t.Fatal("expected nil pick sequence before SetPickSequence")
		}
		if err := s.SetPickSequence(42); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.PickSequence() == nil || *s.PickSequence() != 42 {
			t.Fatalf("unexpected pick sequence %v", s.PickSequence())
		}
	})

	t.Run("rejects a negative sequence", func(t *testing.T) {
		s := mustNewActiveSlot(t)
		if err := s.SetPickSequence(-1); !errors.Is(err, slot.ErrNegativePickSequence) {
			t.Fatalf("expected ErrNegativePickSequence, got %v", err)
		}
	})

	t.Run("rejects setting on a decommissioned slot", func(t *testing.T) {
		s := mustNewActiveSlot(t)
		if err := s.Decommission(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := s.SetPickSequence(1); !errors.Is(err, slot.ErrSlotDecommissioned) {
			t.Fatalf("expected ErrSlotDecommissioned, got %v", err)
		}
	})
}
