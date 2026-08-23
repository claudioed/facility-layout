package usecases

import (
	"context"
	"sort"

	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/domain/slot"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

// ZoneGrid is one Zone's slots shaped as an explicit 2D matrix, ready for a
// UI to iterate and paint as a warehouse map: rows are Levels, columns are
// (Aisle, Bay) pairs in aisle walk order, and a cell holds the slots at
// that coordinate (nil for a gap). No client-side layout maths required.
type ZoneGrid struct {
	Zone    *zone.Zone
	Columns []GridColumn
	Levels  []string
	Rows    []GridRow
}

// GridColumn is one (Aisle, Bay) column of the grid, carrying the aisle's
// walk-order hint so a renderer can label or space columns by travel order.
type GridColumn struct {
	AisleID      string
	AisleCode    string
	Bay          string
	SequenceHint int
}

// GridRow is one Level of the grid. Cells is index-aligned with the grid's
// Columns.
type GridRow struct {
	Level string
	Cells []GridCell
}

// GridCell holds the slots at one (Aisle, Bay, Level) coordinate — usually
// several, one per Position. An empty cell (no Slots) is a gap in the rack.
type GridCell struct {
	Slots []*slot.LocationSlot
}

// GetZoneGrid assembles one Zone's 2D grid. Read-only: no writes, no events.
type GetZoneGrid struct {
	Zones  ports.ZoneRepo
	Aisles ports.AisleRepo
	Slots  ports.SlotRepo
}

// Execute returns the zone's grid, or ErrZoneNotFound.
func (uc *GetZoneGrid) Execute(ctx context.Context, zoneID string) (*ZoneGrid, error) {
	z, err := uc.Zones.FindByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, ErrZoneNotFound
	}

	aisles, err := uc.Aisles.ListByZone(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	sortAislesByWalkOrder(aisles)

	slots, err := uc.Slots.ListByZone(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	// Index the zone's slots by (aisle, bay, level) so building the matrix
	// is a lookup per cell rather than a scan per cell.
	type coordinate struct{ aisle, bay, level string }
	byCoordinate := make(map[coordinate][]*slot.LocationSlot, len(slots))
	baysByAisle := make(map[string]map[string]struct{}, len(aisles))
	levelSet := make(map[string]struct{})

	for _, s := range slots {
		code := s.Code()
		key := coordinate{aisle: code.Aisle(), bay: code.Bay(), level: code.Level()}
		byCoordinate[key] = append(byCoordinate[key], s)

		if _, ok := baysByAisle[code.Aisle()]; !ok {
			baysByAisle[code.Aisle()] = make(map[string]struct{})
		}
		baysByAisle[code.Aisle()][code.Bay()] = struct{}{}
		levelSet[code.Level()] = struct{}{}
	}

	grid := &ZoneGrid{Zone: z, Columns: []GridColumn{}, Levels: sortedKeys(levelSet), Rows: []GridRow{}}

	for _, a := range aisles {
		bays := sortedKeys(baysByAisle[a.AisleCode()])
		for _, bay := range bays {
			grid.Columns = append(grid.Columns, GridColumn{
				AisleID:      a.ID(),
				AisleCode:    a.AisleCode(),
				Bay:          bay,
				SequenceHint: a.SequenceHint(),
			})
		}
	}

	for _, level := range grid.Levels {
		row := GridRow{Level: level, Cells: make([]GridCell, 0, len(grid.Columns))}
		for _, column := range grid.Columns {
			cellSlots := byCoordinate[coordinate{aisle: column.AisleCode, bay: column.Bay, level: level}]
			sortSlotsByCoordinate(cellSlots)
			row.Cells = append(row.Cells, GridCell{Slots: cellSlots})
		}
		grid.Rows = append(grid.Rows, row)
	}

	return grid, nil
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
