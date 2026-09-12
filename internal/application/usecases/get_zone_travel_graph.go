package usecases

import (
	"context"

	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/domain/travel"
)

// GetZoneTravelGraph reads one Zone's travel graph as nodes + edges,
// ready for a caller to render or feed into its own shortest-path
// computation (ADR-0017). It is a read model: no state is stored, no
// event is published, and it is built fresh from the zone's aisles,
// slots, and cross-aisles on every call rather than persisted separately.
type GetZoneTravelGraph struct {
	Zones       ports.ZoneRepo
	Aisles      ports.AisleRepo
	Slots       ports.SlotRepo
	CrossAisles ports.CrossAisleRepo
}

// TravelGraphView is the nodes and edges of a zone's travel graph, shaped
// for direct rendering.
type TravelGraphView struct {
	Nodes []travel.Node
	Edges []travel.Edge
}

// Execute returns the zone's travel graph, or ErrZoneNotFound.
func (uc *GetZoneTravelGraph) Execute(ctx context.Context, zoneID string) (*TravelGraphView, error) {
	graph, err := buildZoneGraph(ctx, uc.Zones, uc.Aisles, uc.Slots, uc.CrossAisles, zoneID)
	if err != nil {
		return nil, err
	}
	return &TravelGraphView{Nodes: graph.AllNodes(), Edges: graph.AllEdges()}, nil
}
