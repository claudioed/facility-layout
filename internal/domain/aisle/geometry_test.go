package aisle_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/shared"
)

func mustAislePoint(t *testing.T, x, y, z float64) shared.Point3D {
	t.Helper()
	p, err := shared.NewPoint3D(x, y, z)
	if err != nil {
		t.Fatalf("point: %v", err)
	}
	return p
}

func mustAisleSegment(t *testing.T) shared.Segment {
	t.Helper()
	s, err := shared.NewSegment(mustAislePoint(t, 0, 0, 0), mustAislePoint(t, 10, 0, 0))
	if err != nil {
		t.Fatalf("segment: %v", err)
	}
	return s
}

func mustNewActiveAisle(t *testing.T) *aisle.Aisle {
	t.Helper()
	a, err := aisle.NewAisle("WH1-STOR-AMB", "A07", 7, shared.TwoWay)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return a
}

func TestAisleSetCentreline(t *testing.T) {
	centreline := mustAisleSegment(t)

	t.Run("sets a real centreline on an active aisle", func(t *testing.T) {
		a := mustNewActiveAisle(t)
		if !a.Centreline().IsZero() {
			t.Fatal("expected no centreline before SetCentreline")
		}
		if err := a.SetCentreline(centreline); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Centreline() != centreline {
			t.Fatalf("unexpected centreline %+v", a.Centreline())
		}
	})

	t.Run("rejects the zero segment", func(t *testing.T) {
		a := mustNewActiveAisle(t)
		if err := a.SetCentreline(shared.Segment{}); err == nil {
			t.Fatal("expected an error for a zero segment")
		}
	})

	t.Run("rejects setting a centreline on a decommissioned aisle", func(t *testing.T) {
		a := mustNewActiveAisle(t)
		if err := a.Decommission(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := a.SetCentreline(centreline); !errors.Is(err, aisle.ErrAisleDecommissioned) {
			t.Fatalf("expected ErrAisleDecommissioned, got %v", err)
		}
	})
}
