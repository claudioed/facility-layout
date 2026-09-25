package aisle_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/aisle"
)

func TestNewCrossAisle(t *testing.T) {
	t.Run("constructs an active cross-aisle connecting two distinct aisles", func(t *testing.T) {
		c, err := aisle.NewCrossAisle("WH1-STOR-AMB", "A07", "A09", "03")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.ZoneID() != "WH1-STOR-AMB" || c.FromAisle() != "A07" || c.ToAisle() != "A09" || c.AtBay() != "03" {
			t.Fatalf("unexpected cross-aisle %+v", c)
		}
		if !c.IsActive() {
			t.Fatal("a newly constructed cross-aisle must be active")
		}
	})

	t.Run("rejects an empty zone id", func(t *testing.T) {
		if _, err := aisle.NewCrossAisle("", "A07", "A09", "03"); !errors.Is(err, aisle.ErrCrossAisleEmptyZoneID) {
			t.Fatalf("expected ErrCrossAisleEmptyZoneID, got %v", err)
		}
	})

	t.Run("rejects an empty from-aisle", func(t *testing.T) {
		if _, err := aisle.NewCrossAisle("WH1-STOR-AMB", "", "A09", "03"); !errors.Is(err, aisle.ErrCrossAisleEmptyFromAisle) {
			t.Fatalf("expected ErrCrossAisleEmptyFromAisle, got %v", err)
		}
	})

	t.Run("rejects an empty to-aisle", func(t *testing.T) {
		if _, err := aisle.NewCrossAisle("WH1-STOR-AMB", "A07", "", "03"); !errors.Is(err, aisle.ErrCrossAisleEmptyToAisle) {
			t.Fatalf("expected ErrCrossAisleEmptyToAisle, got %v", err)
		}
	})

	t.Run("rejects connecting an aisle to itself", func(t *testing.T) {
		if _, err := aisle.NewCrossAisle("WH1-STOR-AMB", "A07", "A07", "03"); !errors.Is(err, aisle.ErrCrossAisleSameAisle) {
			t.Fatalf("expected ErrCrossAisleSameAisle, got %v", err)
		}
	})

	t.Run("rejects an empty bay", func(t *testing.T) {
		if _, err := aisle.NewCrossAisle("WH1-STOR-AMB", "A07", "A09", ""); !errors.Is(err, aisle.ErrCrossAisleEmptyBay) {
			t.Fatalf("expected ErrCrossAisleEmptyBay, got %v", err)
		}
	})
}

func TestCrossAisleDecommission(t *testing.T) {
	c, err := aisle.NewCrossAisle("WH1-STOR-AMB", "A07", "A09", "03")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.Decommission(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.IsActive() {
		t.Fatal("expected the cross-aisle to stop being active")
	}
	if err := c.Decommission(); !errors.Is(err, aisle.ErrCrossAisleAlreadyDecommissioned) {
		t.Fatalf("expected ErrCrossAisleAlreadyDecommissioned, got %v", err)
	}
}

func TestCrossAisleConnects(t *testing.T) {
	c, err := aisle.NewCrossAisle("WH1-STOR-AMB", "A07", "A09", "03")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("returns the other aisle from the from-side", func(t *testing.T) {
		other, ok := c.Connects("A07")
		if !ok || other != "A09" {
			t.Fatalf("expected (A09, true), got (%q, %v)", other, ok)
		}
	})

	t.Run("returns the other aisle from the to-side", func(t *testing.T) {
		other, ok := c.Connects("A09")
		if !ok || other != "A07" {
			t.Fatalf("expected (A07, true), got (%q, %v)", other, ok)
		}
	})

	t.Run("reports false for an unrelated aisle", func(t *testing.T) {
		if _, ok := c.Connects("A99"); ok {
			t.Fatal("expected Connects to report false for an unrelated aisle")
		}
	})
}

func TestRehydrateCrossAisle(t *testing.T) {
	t.Run("active", func(t *testing.T) {
		c := aisle.RehydrateCrossAisle("WH1-STOR-AMB", "A07", "A09", "03", false)
		if !c.IsActive() {
			t.Fatal("expected an active cross-aisle")
		}
	})

	t.Run("decommissioned", func(t *testing.T) {
		c := aisle.RehydrateCrossAisle("WH1-STOR-AMB", "A07", "A09", "03", true)
		if c.IsActive() {
			t.Fatal("expected a decommissioned cross-aisle")
		}
		if err := c.Decommission(); !errors.Is(err, aisle.ErrCrossAisleAlreadyDecommissioned) {
			t.Fatalf("expected ErrCrossAisleAlreadyDecommissioned, got %v", err)
		}
	})
}
