package shared_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/shared"
)

func TestParseTemperatureClass(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    shared.TemperatureClass
		wantErr error
	}{
		{name: "ambient", raw: "Ambient", want: shared.Ambient},
		{name: "chilled", raw: "Chilled", want: shared.Chilled},
		{name: "frozen", raw: "Frozen", want: shared.Frozen},
		{name: "unknown", raw: "Tepid", wantErr: shared.ErrUnknownTemperatureClass},
		{name: "wrong case", raw: "frozen", wantErr: shared.ErrUnknownTemperatureClass},
		{name: "empty", raw: "", wantErr: shared.ErrUnknownTemperatureClass},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shared.ParseTemperatureClass(tc.raw)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestParseDirection(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    shared.Direction
		wantErr error
	}{
		{name: "one way", raw: "OneWay", want: shared.OneWay},
		{name: "two way", raw: "TwoWay", want: shared.TwoWay},
		{name: "unknown", raw: "Sideways", wantErr: shared.ErrUnknownDirection},
		{name: "empty", raw: "", wantErr: shared.ErrUnknownDirection},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shared.ParseDirection(tc.raw)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    shared.Status
		wantErr error
	}{
		{name: "active", raw: "Active", want: shared.Active},
		{name: "under maintenance", raw: "UnderMaintenance", want: shared.UnderMaintenance},
		{name: "decommissioned", raw: "Decommissioned", want: shared.Decommissioned},
		{name: "unknown", raw: "Retired", wantErr: shared.ErrUnknownStatus},
		{name: "empty", raw: "", wantErr: shared.ErrUnknownStatus},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shared.ParseStatus(tc.raw)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
