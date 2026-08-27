// Package report holds the facility-layout "Layout Catalog Growth & Change"
// read model: the shapes of the analytical report the data product serves, the
// query that selects it, and the outbound ports the writer and reader adapters
// implement. It is a read-model region that depends on nothing else in this
// module — the OLTP domain and application layers must not import it, and it
// must not import them (ADR-0010).
//
// The report tracks how the warehouse map grows and changes over time. Because
// the layout is a slow-changing reference catalog (a Generic Subdomain the rest
// of the estate conforms to, not a live transactional stream), rows are bucketed
// by DAY rather than by hour.
package report

import "time"

// Granularity is the time-bucket resolution a report is rolled up to. The
// layout catalog changes slowly, so only daily buckets are modelled.
type Granularity string

const (
	// GranularityDay rolls rows up into UTC day buckets (midnight UTC).
	GranularityDay Granularity = "day"
)

// RowKey identifies a single catalog-growth row: the scope the change applies
// to and the UTC day bucket the row aggregates.
//
// Scope is the part of the warehouse map a change belongs to:
//   - a SiteCode for site- and zone-level changes (a zone is added to a site),
//   - a ZoneID for aisle- and slot-level changes (they live inside a zone),
//   - the empty string for catalog-wide definitions that are not scoped to any
//     one site or zone: LocationType registrations, PlacementRule definitions,
//     and bulk imports.
//
// DayBucket is the bucket start, truncated to the UTC day.
type RowKey struct {
	Scope     string
	DayBucket time.Time
}

// Row is one aggregated catalog-growth row for a (scope, dayBucket) key. Each
// counter tracks how many of a given catalog-change event landed in the bucket;
// the import fields carry the bulk-import row tallies.
type Row struct {
	Key RowKey
	// SlotsRegistered is the number of LocationSlotRegistered events.
	SlotsRegistered int
	// SlotsDecommissioned is the number of LocationSlotDecommissioned events.
	SlotsDecommissioned int
	// ZonesRegistered is the number of ZoneRegistered events.
	ZonesRegistered int
	// AislesRegistered is the number of AisleRegistered events.
	AislesRegistered int
	// LocationTypesRegistered is the number of LocationTypeRegistered events.
	LocationTypesRegistered int
	// PlacementRulesDefined is the number of PlacementRuleDefined events.
	PlacementRulesDefined int
	// SitesRegistered is the number of SiteRegistered events.
	SitesRegistered int
	// BulkImports is the number of FacilityLayoutImported events.
	BulkImports int
	// ImportRowsSubmitted/Imported/Rejected accumulate the row tallies each
	// FacilityLayoutImported event carries.
	ImportRowsSubmitted int
	ImportRowsImported  int
	ImportRowsRejected  int
}

// CatalogReport is the full result of a report query: the matching rows.
type CatalogReport struct {
	Rows []Row
}

// ReportQuery selects and filters the rows a report covers. From is inclusive
// and To is exclusive, both compared against a row's DayBucket. Scope is an
// optional exact-match filter (empty means "no filter on scope" — note this is
// distinct from a row whose scope is itself the empty catalog-wide scope, which
// an empty filter still returns).
type ReportQuery struct {
	From        time.Time
	To          time.Time
	Scope       string
	Granularity Granularity
}
