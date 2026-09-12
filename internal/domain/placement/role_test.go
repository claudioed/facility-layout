package placement_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/placement"
)

func TestParseLocationRole(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    placement.LocationRole
		wantErr error
	}{
		{name: "storage", raw: "Storage", want: placement.Storage},
		{name: "dock", raw: "Dock", want: placement.Dock},
		{name: "yard", raw: "Yard", want: placement.Yard},
		{name: "work center", raw: "WorkCenter", want: placement.WorkCenter},
		{name: "drop", raw: "Drop", want: placement.Drop},
		{name: "staging", raw: "Staging", want: placement.RoleStaging},
		{name: "qc", raw: "QC", want: placement.QC},
		{name: "consolidation", raw: "Consolidation", want: placement.Consolidation},
		{name: "shipping", raw: "Shipping", want: placement.Shipping},
		{name: "unknown", raw: "Warehouse", wantErr: placement.ErrUnknownLocationRole},
		{name: "empty", raw: "", wantErr: placement.ErrUnknownLocationRole},
		{name: "wrong case", raw: "dock", wantErr: placement.ErrUnknownLocationRole},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := placement.ParseLocationRole(tc.raw)
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

func TestLocationRoleRequiresCapacity(t *testing.T) {
	tests := []struct {
		role placement.LocationRole
		want bool
	}{
		{placement.Storage, true},
		{placement.RoleStaging, true},
		{placement.Drop, true},
		{placement.Consolidation, true},
		{placement.Dock, false},
		{placement.Yard, false},
		{placement.WorkCenter, false},
		{placement.QC, false},
		{placement.Shipping, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.role), func(t *testing.T) {
			if got := tc.role.RequiresCapacity(); got != tc.want {
				t.Fatalf("expected RequiresCapacity()=%t for %q, got %t", tc.want, tc.role, got)
			}
		})
	}
}
