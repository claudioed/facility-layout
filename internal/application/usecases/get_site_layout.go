package usecases

import (
	"context"
	"sort"

	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/site"
	"github.com/claudioed/facility-layout/internal/domain/slot"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

// SiteLayout is the readable, drawable projection of one Site's whole
// structure: zones -> aisles -> location slots. It is a PROJECTION built by
// querying across the aggregates, never separately stored state.
type SiteLayout struct {
	Site  *site.Site
	Zones []ZoneLayout
}

// ZoneLayout is one Zone and every Aisle inside it.
type ZoneLayout struct {
	Zone   *zone.Zone
	Aisles []AisleLayout
}

// AisleLayout is one Aisle and every LocationSlot inside it, ordered
// bay -> level -> position so a client can paint it left-to-right,
// bottom-to-top without sorting.
type AisleLayout struct {
	Aisle *aisle.Aisle
	Slots []*slot.LocationSlot
}

// GetSiteLayout assembles the full nested structure of one Site. Read-only:
// no writes, no events.
type GetSiteLayout struct {
	Sites  ports.SiteRepo
	Zones  ports.ZoneRepo
	Aisles ports.AisleRepo
	Slots  ports.SlotRepo
}

// Execute returns the site's full layout, or ErrSiteNotFound.
func (uc *GetSiteLayout) Execute(ctx context.Context, siteCode string) (*SiteLayout, error) {
	s, err := uc.Sites.FindByCode(ctx, siteCode)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, ErrSiteNotFound
	}

	zones, err := uc.Zones.ListBySite(ctx, siteCode)
	if err != nil {
		return nil, err
	}
	sort.Slice(zones, func(i, j int) bool { return zones[i].ID() < zones[j].ID() })

	layout := &SiteLayout{Site: s, Zones: make([]ZoneLayout, 0, len(zones))}
	for _, z := range zones {
		aisles, err := uc.Aisles.ListByZone(ctx, z.ID())
		if err != nil {
			return nil, err
		}
		sortAislesByWalkOrder(aisles)

		zoneLayout := ZoneLayout{Zone: z, Aisles: make([]AisleLayout, 0, len(aisles))}
		for _, a := range aisles {
			slots, err := uc.Slots.ListByAisle(ctx, a.ID())
			if err != nil {
				return nil, err
			}
			sortSlotsByCoordinate(slots)
			zoneLayout.Aisles = append(zoneLayout.Aisles, AisleLayout{Aisle: a, Slots: slots})
		}
		layout.Zones = append(layout.Zones, zoneLayout)
	}
	return layout, nil
}

// sortAislesByWalkOrder orders aisles by SequenceHint — their walk-order
// position — falling back to the aisle code so the order is total and
// stable even when two aisles share a hint.
func sortAislesByWalkOrder(aisles []*aisle.Aisle) {
	sort.Slice(aisles, func(i, j int) bool {
		if aisles[i].SequenceHint() != aisles[j].SequenceHint() {
			return aisles[i].SequenceHint() < aisles[j].SequenceHint()
		}
		return aisles[i].AisleCode() < aisles[j].AisleCode()
	})
}

// sortSlotsByCoordinate orders slots bay -> level -> position.
func sortSlotsByCoordinate(slots []*slot.LocationSlot) {
	sort.Slice(slots, func(i, j int) bool {
		left, right := slots[i].Code(), slots[j].Code()
		if left.Bay() != right.Bay() {
			return left.Bay() < right.Bay()
		}
		if left.Level() != right.Level() {
			return left.Level() < right.Level()
		}
		return left.Position() < right.Position()
	})
}
