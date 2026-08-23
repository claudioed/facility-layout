package usecases_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/placement"
)

func isErr(err, want error) bool { return errors.Is(err, want) }

// decommissionSite retires a site directly through the repository: v1
// exposes no HTTP-facing site-decommission use case, but the invariant
// "no structure may be registered against a non-Active parent" still has to
// be provable.
func decommissionSite(t *testing.T, h *harness, code string) {
	t.Helper()
	s, err := h.sites.FindByCode(h.ctx(), code)
	if err != nil || s == nil {
		t.Fatalf("seeding: site %q not found (%v)", code, err)
	}
	if err := s.Decommission(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.sites.Save(h.ctx(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func decommissionZone(t *testing.T, h *harness, id string) {
	t.Helper()
	z, err := h.zones.FindByID(h.ctx(), id)
	if err != nil || z == nil {
		t.Fatalf("seeding: zone %q not found (%v)", id, err)
	}
	if err := z.Decommission(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.zones.Save(h.ctx(), z); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func decommissionAisle(t *testing.T, h *harness, id string) {
	t.Helper()
	a, err := h.aisles.FindByID(h.ctx(), id)
	if err != nil || a == nil {
		t.Fatalf("seeding: aisle %q not found (%v)", id, err)
	}
	if err := a.Decommission(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.aisles.Save(h.ctx(), a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustDefineRule(t *testing.T, h *harness, id, locationType string, effect placement.Effect, predicate placement.ZonePredicate) {
	t.Helper()
	if _, err := h.definePlacementRule.Execute(h.ctx(), id, locationType, effect, predicate); err != nil {
		t.Fatalf("seeding rule %q: %v", id, err)
	}
}
