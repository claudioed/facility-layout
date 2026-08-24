// Package mcp is the inbound Model Context Protocol adapter: it exposes this
// bounded context to the AI ecosystem as a second driving adapter over the
// same application-layer use cases the HTTP adapter uses. It is built on the
// official MCP Go SDK and served over Streamable HTTP.
//
// Per ADR-0007 and the MCP governance charter, this package depends inward on
// the application layer (use cases and ports) and the domain only — never on
// an outbound adapter. The composition root (cmd/mcp) wires concrete
// repositories into the use cases and query port. Tool handlers call use
// cases; domain structs never leak across the tool boundary.
//
// facility-layout is a read-only Open Host Service: its warehouse map is
// consumed, never mutated, by the rest of the estate. This server therefore
// registers only read tools, a scoped resource, and a prompt — no write tool.
// The read/read-write Scope seam in auth.go is kept identical to the pilot so
// a future write tool needs no auth rework, but nothing here registers one.
package mcp

import (
	"github.com/claudioed/facility-layout/internal/application/usecases"
	"github.com/claudioed/facility-layout/internal/domain/site"
	"github.com/claudioed/facility-layout/internal/domain/slot"
)

// tool-boundary DTOs -----------------------------------------------------------
//
// These are the ONLY shapes that cross the tool boundary. Domain structs
// (site.Site, zone.Zone, aisle.Aisle, slot.LocationSlot) never leave this
// package: they are mapped into the compact, bounded DTOs below, sized for an
// agent to reason over rather than for completeness.

// siteRef is the compact identity of a Site: its code and human name. It is
// the entry in list_sites and the header of a site layout.
type siteRef struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// siteLayoutDTO is the compact, nested projection of one Site's structure —
// zones -> aisles -> slot codes — returned by get_site_layout. It is a bounded
// view: each slot is reduced to its code string, not the full aggregate, so
// the payload stays scoped to "what is the shape of this site" rather than
// dumping every slot's capacity envelope.
type siteLayoutDTO struct {
	Site  siteRef         `json:"site"`
	Zones []zoneLayoutDTO `json:"zones"`
}

// zoneLayoutDTO is one Zone and its aisles within a site layout.
type zoneLayoutDTO struct {
	ZoneID           string           `json:"zoneId"`
	AreaCode         string           `json:"areaCode"`
	ZoneCode         string           `json:"zoneCode"`
	TemperatureClass string           `json:"temperatureClass"`
	Hazmat           bool             `json:"hazmat"`
	Aisles           []aisleLayoutDTO `json:"aisles"`
}

// aisleLayoutDTO is one Aisle and the codes of the slots inside it, in
// walk/coordinate order. Slots are their location-code strings only — the
// intent-level "map", not the full slot records.
type aisleLayoutDTO struct {
	AisleID      string   `json:"aisleId"`
	AisleCode    string   `json:"aisleCode"`
	SequenceHint int      `json:"sequenceHint"`
	Direction    string   `json:"direction"`
	SlotCodes    []string `json:"slotCodes"`
}

// zoneGridDTO is the compact projection of one Zone's drawable grid, returned
// by get_zone_grid. Rows are Levels, columns are (Aisle, Bay) pairs in walk
// order, and each cell holds the location-code strings at that coordinate.
type zoneGridDTO struct {
	ZoneID  string          `json:"zoneId"`
	Columns []gridColumnDTO `json:"columns"`
	Levels  []string        `json:"levels"`
	Rows    []gridRowDTO    `json:"rows"`
}

// gridColumnDTO is one (Aisle, Bay) column of the grid.
type gridColumnDTO struct {
	AisleID      string `json:"aisleId"`
	AisleCode    string `json:"aisleCode"`
	Bay          string `json:"bay"`
	SequenceHint int    `json:"sequenceHint"`
}

// gridRowDTO is one Level of the grid; Cells is index-aligned with Columns.
type gridRowDTO struct {
	Level string        `json:"level"`
	Cells []gridCellDTO `json:"cells"`
}

// gridCellDTO holds the slot codes at one (Aisle, Bay, Level) coordinate.
// An empty SlotCodes is a gap in the rack.
type gridCellDTO struct {
	SlotCodes []string `json:"slotCodes"`
}

// mapping functions ------------------------------------------------------------

// toSiteRef maps a domain Site to its compact reference DTO.
func toSiteRef(s *site.Site) siteRef {
	return siteRef{Code: s.Code(), Name: s.Name()}
}

// toSiteLayoutDTO maps the GetSiteLayout read model into the bounded,
// nested tool DTO. Nothing but this file's DTOs crosses the tool boundary.
func toSiteLayoutDTO(layout *usecases.SiteLayout) siteLayoutDTO {
	out := siteLayoutDTO{Site: toSiteRef(layout.Site), Zones: make([]zoneLayoutDTO, 0, len(layout.Zones))}
	for _, zl := range layout.Zones {
		z := zoneLayoutDTO{
			ZoneID:           zl.Zone.ID(),
			AreaCode:         zl.Zone.AreaCode(),
			ZoneCode:         zl.Zone.ZoneCode(),
			TemperatureClass: string(zl.Zone.TemperatureClass()),
			Hazmat:           zl.Zone.Hazmat(),
			Aisles:           make([]aisleLayoutDTO, 0, len(zl.Aisles)),
		}
		for _, al := range zl.Aisles {
			z.Aisles = append(z.Aisles, aisleLayoutDTO{
				AisleID:      al.Aisle.ID(),
				AisleCode:    al.Aisle.AisleCode(),
				SequenceHint: al.Aisle.SequenceHint(),
				Direction:    string(al.Aisle.Direction()),
				SlotCodes:    slotCodes(al.Slots),
			})
		}
		out.Zones = append(out.Zones, z)
	}
	return out
}

// toZoneGridDTO maps the GetZoneGrid read model into the compact grid DTO.
func toZoneGridDTO(grid *usecases.ZoneGrid) zoneGridDTO {
	out := zoneGridDTO{
		ZoneID:  grid.Zone.ID(),
		Columns: make([]gridColumnDTO, 0, len(grid.Columns)),
		Levels:  grid.Levels,
		Rows:    make([]gridRowDTO, 0, len(grid.Rows)),
	}
	for _, c := range grid.Columns {
		out.Columns = append(out.Columns, gridColumnDTO{
			AisleID:      c.AisleID,
			AisleCode:    c.AisleCode,
			Bay:          c.Bay,
			SequenceHint: c.SequenceHint,
		})
	}
	for _, r := range grid.Rows {
		row := gridRowDTO{Level: r.Level, Cells: make([]gridCellDTO, 0, len(r.Cells))}
		for _, cell := range r.Cells {
			row.Cells = append(row.Cells, gridCellDTO{SlotCodes: slotCodes(cell.Slots)})
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// slotCodes reduces a list of LocationSlot aggregates to their code strings,
// the only slot detail the map-level tools expose.
func slotCodes(slots []*slot.LocationSlot) []string {
	codes := make([]string, 0, len(slots))
	for _, s := range slots {
		codes = append(codes, s.Code().String())
	}
	return codes
}
