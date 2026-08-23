package shared_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/shared"
)

func TestNewCapacity(t *testing.T) {
	tests := []struct {
		name    string
		weight  float64
		volume  float64
		wantErr error
	}{
		{name: "a pallet-rack envelope", weight: 1200, volume: 2.4},
		{name: "zero weight", weight: 0, volume: 2.4, wantErr: shared.ErrInvalidMaxWeight},
		{name: "negative weight", weight: -1, volume: 2.4, wantErr: shared.ErrInvalidMaxWeight},
		{name: "zero volume", weight: 1200, volume: 0, wantErr: shared.ErrInvalidMaxVolume},
		{name: "negative volume", weight: 1200, volume: -0.5, wantErr: shared.ErrInvalidMaxVolume},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capacity, err := shared.NewCapacity(tc.weight, tc.volume)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				if !capacity.IsZero() {
					t.Fatal("expected the zero Capacity on failure")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if capacity.MaxWeightKg() != tc.weight || capacity.MaxVolumeM3() != tc.volume {
				t.Fatalf("expected %v/%v, got %v/%v", tc.weight, tc.volume, capacity.MaxWeightKg(), capacity.MaxVolumeM3())
			}
			if capacity.IsZero() {
				t.Fatal("a valid capacity must never be the zero value")
			}
		})
	}
}
