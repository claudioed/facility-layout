package http

import (
	"github.com/claudioed/facility-layout/internal/application/usecases"
	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/site"
	"github.com/claudioed/facility-layout/internal/domain/slot"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

func toSiteResponse(s *site.Site) siteResponse {
	return siteResponse{SiteCode: s.Code(), Name: s.Name(), Status: string(s.Status())}
}

func toZoneResponse(z *zone.Zone) zoneResponse {
	return zoneResponse{
		ZoneID:           z.ID(),
		SiteCode:         z.SiteCode(),
		AreaCode:         z.AreaCode(),
		ZoneCode:         z.ZoneCode(),
		TemperatureClass: string(z.TemperatureClass()),
		Hazmat:           z.Hazmat(),
		Status:           string(z.Status()),
	}
}

func toAisleResponse(a *aisle.Aisle) aisleResponse {
	return aisleResponse{
		AisleID:      a.ID(),
		ZoneID:       a.ZoneID(),
		AisleCode:    a.AisleCode(),
		SequenceHint: a.SequenceHint(),
		Direction:    string(a.Direction()),
		Status:       string(a.Status()),
	}
}

func toCapacityResponse(c shared.Capacity) capacityResponse {
	return capacityResponse{MaxWeightKg: c.MaxWeightKg(), MaxVolumeM3: c.MaxVolumeM3()}
}

func toLocationTypeResponse(t placement.LocationType) locationTypeResponse {
	return locationTypeResponse{Name: t.Name(), DefaultCapacity: toCapacityResponse(t.DefaultCapacity())}
}

func toPlacementRuleResponse(rule placement.PlacementRule) placementRuleResponse {
	predicate := rule.Predicate()
	return placementRuleResponse{
		RuleID:       rule.ID(),
		LocationType: rule.LocationType(),
		Effect:       string(rule.Effect()),
		Zone: zonePredicateResponse{
			ZoneCode:         predicate.ZoneCode(),
			TemperatureClass: string(predicate.TemperatureClass()),
			Hazmat:           predicate.Hazmat(),
		},
		Description: rule.Describe(),
	}
}

func toCoordinatesResponse(code shared.LocationCode) coordinatesResponse {
	return coordinatesResponse{
		Site:     code.Site(),
		Area:     code.Area(),
		Zone:     code.Zone(),
		Aisle:    code.Aisle(),
		Bay:      code.Bay(),
		Level:    code.Level(),
		Position: code.Position(),
	}
}

func toLocationSlotResponse(s *slot.LocationSlot) locationSlotResponse {
	code := s.Code()
	return locationSlotResponse{
		LocationCode: code.String(),
		ZoneID:       code.ZoneID(),
		AisleID:      code.AisleID(),
		Coordinates:  toCoordinatesResponse(code),
		LocationType: s.LocationType(),
		Capacity:     toCapacityResponse(s.Capacity()),
		Status:       string(s.Status()),
	}
}

func toLocationClassificationResponse(z *zone.Zone) locationClassificationResponse {
	return locationClassificationResponse{
		Hazmat:           z.Hazmat(),
		TemperatureClass: string(z.TemperatureClass()),
	}
}

func toImportRow(row importRowRequest) usecases.ImportRow {
	out := usecases.ImportRow{
		SiteCode:         row.SiteCode,
		SiteName:         row.SiteName,
		AreaCode:         row.AreaCode,
		ZoneCode:         row.ZoneCode,
		TemperatureClass: shared.TemperatureClass(row.TemperatureClass),
		Hazmat:           row.Hazmat,
		AisleCode:        row.AisleCode,
		SequenceHint:     row.SequenceHint,
		Direction:        shared.Direction(row.Direction),
		Bay:              row.Bay,
		Level:            row.Level,
		Position:         row.Position,
		LocationType:     row.LocationType,
	}
	if row.CapacityOverride != nil {
		out.MaxWeightKg = row.CapacityOverride.MaxWeightKg
		out.MaxVolumeM3 = row.CapacityOverride.MaxVolumeM3
	}
	return out
}

func toImportReportResponse(report *usecases.ImportReport) importReportResponse {
	results := make([]importRowResultResponse, 0, len(report.Results))
	for _, result := range report.Results {
		results = append(results, importRowResultResponse{
			Index:        result.Index,
			LocationCode: result.LocationCode,
			Succeeded:    result.Succeeded,
			Error:        result.Error,
		})
	}
	return importReportResponse{
		RowsSubmitted: report.RowsSubmitted,
		SlotsImported: report.SlotsImported,
		RowsRejected:  report.RowsRejected,
		Results:       results,
	}
}

func toSiteLayoutResponse(layout *usecases.SiteLayout) siteLayoutResponse {
	out := siteLayoutResponse{Site: toSiteResponse(layout.Site), Zones: make([]zoneLayoutResponse, 0, len(layout.Zones))}

	for _, zoneLayout := range layout.Zones {
		zoneOut := zoneLayoutResponse{
			zoneResponse: toZoneResponse(zoneLayout.Zone),
			Aisles:       make([]aisleLayoutResponse, 0, len(zoneLayout.Aisles)),
		}
		for _, aisleLayout := range zoneLayout.Aisles {
			slots := make([]locationSlotResponse, 0, len(aisleLayout.Slots))
			for _, s := range aisleLayout.Slots {
				slots = append(slots, toLocationSlotResponse(s))
			}
			zoneOut.Aisles = append(zoneOut.Aisles, aisleLayoutResponse{
				aisleResponse: toAisleResponse(aisleLayout.Aisle),
				Slots:         slots,
			})
			out.Totals.Aisles++
			out.Totals.Slots += len(slots)
		}
		out.Zones = append(out.Zones, zoneOut)
		out.Totals.Zones++
	}
	return out
}

func toZoneGridResponse(grid *usecases.ZoneGrid) zoneGridResponse {
	out := zoneGridResponse{
		Zone:    toZoneResponse(grid.Zone),
		Columns: make([]gridColumnResponse, 0, len(grid.Columns)),
		Levels:  grid.Levels,
		Rows:    make([]gridRowResponse, 0, len(grid.Rows)),
	}
	if out.Levels == nil {
		out.Levels = []string{}
	}

	for _, column := range grid.Columns {
		out.Columns = append(out.Columns, gridColumnResponse{
			AisleID:      column.AisleID,
			AisleCode:    column.AisleCode,
			Bay:          column.Bay,
			SequenceHint: column.SequenceHint,
		})
	}

	for _, row := range grid.Rows {
		cells := make([]*gridCellResponse, 0, len(row.Cells))
		for _, cell := range row.Cells {
			if len(cell.Slots) == 0 {
				// A gap in the rack: JSON null, per CLAUDE.md's grid contract.
				cells = append(cells, nil)
				continue
			}
			positions := make([]gridPositionResponse, 0, len(cell.Slots))
			for _, s := range cell.Slots {
				positions = append(positions, gridPositionResponse{
					LocationCode: s.Code().String(),
					Position:     s.Code().Position(),
					LocationType: s.LocationType(),
					Status:       string(s.Status()),
				})
			}
			cells = append(cells, &gridCellResponse{Positions: positions})
		}
		out.Rows = append(out.Rows, gridRowResponse{Level: row.Level, Cells: cells})
	}
	return out
}
