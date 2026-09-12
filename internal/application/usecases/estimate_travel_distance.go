package usecases

import (
	"context"

	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/travel"
)

// EstimateTravelDistance computes the shortest travel distance between two
// coded locations over the pure-domain travel graph (ADR-0017). It is a
// read model: no state is stored, no event is published. This context
// only reports the map's TOPOLOGY — travel TIME, congestion, and route
// choice under load remain wes-work-planning's concern.
type EstimateTravelDistance struct {
	Zones       ports.ZoneRepo
	Aisles      ports.AisleRepo
	Slots       ports.SlotRepo
	CrossAisles ports.CrossAisleRepo
}

// TravelDistance is the outcome of a distance query: the total length in
// metres, whether any leg of the route was estimated rather than measured,
// and the ordered aisle/bay waypoints traversed.
type TravelDistance struct {
	MetresM   float64
	Estimated bool
	Route     []travel.Node
}

// Execute resolves from and to to their zones and slots, refusing
// (ErrNoRouteBetweenZones) rather than guessing when they are in different
// zones — this phase's graph does not connect zones. Both locations must
// exist (ErrLocationSlotNotFound).
func (uc *EstimateTravelDistance) Execute(ctx context.Context, from, to shared.LocationCode) (*TravelDistance, error) {
	fromSlot, err := uc.Slots.FindByCode(ctx, from)
	if err != nil {
		return nil, err
	}
	if fromSlot == nil {
		return nil, ErrLocationSlotNotFound
	}
	toSlot, err := uc.Slots.FindByCode(ctx, to)
	if err != nil {
		return nil, err
	}
	if toSlot == nil {
		return nil, ErrLocationSlotNotFound
	}

	if from.ZoneID() != to.ZoneID() {
		return nil, ErrNoRouteBetweenZones
	}

	graph, err := buildZoneGraph(ctx, uc.Zones, uc.Aisles, uc.Slots, uc.CrossAisles, from.ZoneID())
	if err != nil {
		return nil, err
	}

	fromNode := travel.Node{AisleID: from.AisleID(), Bay: from.Bay()}
	toNode := travel.Node{AisleID: to.AisleID(), Bay: to.Bay()}
	route, err := graph.Distance(fromNode, toNode)
	if err != nil {
		return nil, err
	}
	return &TravelDistance{MetresM: route.MetresM, Estimated: route.Estimated, Route: route.Nodes}, nil
}
