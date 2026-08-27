// Package http is the inbound chi adapter: DTOs, handlers, routing, and
// domain-error-to-HTTP-status mapping. Domain structs never cross this
// boundary — every response here is a DTO defined in this file.
package http

// ------------------------------------------------------------- requests ----

type registerSiteRequest struct {
	SiteCode string `json:"siteCode"`
	Name     string `json:"name"`
}

type registerZoneRequest struct {
	AreaCode         string `json:"areaCode"`
	ZoneCode         string `json:"zoneCode"`
	TemperatureClass string `json:"temperatureClass"`
	Hazmat           bool   `json:"hazmat"`
}

type registerAisleRequest struct {
	AisleCode    string `json:"aisleCode"`
	SequenceHint int    `json:"sequenceHint"`
	Direction    string `json:"direction"`
}

type capacityRequest struct {
	MaxWeightKg float64 `json:"maxWeightKg"`
	MaxVolumeM3 float64 `json:"maxVolumeM3"`
}

type registerLocationTypeRequest struct {
	Name            string          `json:"name"`
	DefaultCapacity capacityRequest `json:"defaultCapacity"`
}

type zonePredicateRequest struct {
	ZoneCode         string `json:"zoneCode,omitempty"`
	TemperatureClass string `json:"temperatureClass,omitempty"`
	Hazmat           *bool  `json:"hazmat,omitempty"`
}

type definePlacementRuleRequest struct {
	RuleID       string               `json:"ruleId"`
	LocationType string               `json:"locationType"`
	Effect       string               `json:"effect"`
	Zone         zonePredicateRequest `json:"zone"`
}

type registerLocationSlotRequest struct {
	LocationCode     string           `json:"locationCode"`
	LocationType     string           `json:"locationType"`
	CapacityOverride *capacityRequest `json:"capacityOverride,omitempty"`
}

type importRowRequest struct {
	SiteCode         string           `json:"siteCode"`
	SiteName         string           `json:"siteName,omitempty"`
	AreaCode         string           `json:"areaCode"`
	ZoneCode         string           `json:"zoneCode"`
	TemperatureClass string           `json:"temperatureClass"`
	Hazmat           bool             `json:"hazmat"`
	AisleCode        string           `json:"aisleCode"`
	SequenceHint     int              `json:"sequenceHint"`
	Direction        string           `json:"direction,omitempty"`
	Bay              string           `json:"bay"`
	Level            string           `json:"level"`
	Position         string           `json:"position"`
	LocationType     string           `json:"locationType"`
	CapacityOverride *capacityRequest `json:"capacityOverride,omitempty"`
}

// ------------------------------------------------------------ responses ----

type siteResponse struct {
	SiteCode string `json:"siteCode"`
	Name     string `json:"name"`
	Status   string `json:"status"`
}

type zoneResponse struct {
	ZoneID           string `json:"zoneId"`
	SiteCode         string `json:"siteCode"`
	AreaCode         string `json:"areaCode"`
	ZoneCode         string `json:"zoneCode"`
	TemperatureClass string `json:"temperatureClass"`
	Hazmat           bool   `json:"hazmat"`
	Status           string `json:"status"`
}

type aisleResponse struct {
	AisleID      string `json:"aisleId"`
	ZoneID       string `json:"zoneId"`
	AisleCode    string `json:"aisleCode"`
	SequenceHint int    `json:"sequenceHint"`
	Direction    string `json:"direction"`
	Status       string `json:"status"`
}

type capacityResponse struct {
	MaxWeightKg float64 `json:"maxWeightKg"`
	MaxVolumeM3 float64 `json:"maxVolumeM3"`
}

type locationTypeResponse struct {
	Name            string           `json:"name"`
	DefaultCapacity capacityResponse `json:"defaultCapacity"`
}

type placementRuleResponse struct {
	RuleID       string                `json:"ruleId"`
	LocationType string                `json:"locationType"`
	Effect       string                `json:"effect"`
	Zone         zonePredicateResponse `json:"zone"`
	Description  string                `json:"description"`
}

type zonePredicateResponse struct {
	ZoneCode         string `json:"zoneCode,omitempty"`
	TemperatureClass string `json:"temperatureClass,omitempty"`
	Hazmat           *bool  `json:"hazmat,omitempty"`
}

// coordinatesResponse is the exploded LocationCode: the seven typed
// segments a renderer needs to position a slot without re-parsing the code.
type coordinatesResponse struct {
	Site     string `json:"site"`
	Area     string `json:"area"`
	Zone     string `json:"zone"`
	Aisle    string `json:"aisle"`
	Bay      string `json:"bay"`
	Level    string `json:"level"`
	Position string `json:"position"`
}

type locationSlotResponse struct {
	LocationCode string              `json:"locationCode"`
	ZoneID       string              `json:"zoneId"`
	AisleID      string              `json:"aisleId"`
	Coordinates  coordinatesResponse `json:"coordinates"`
	LocationType string              `json:"locationType"`
	Capacity     capacityResponse    `json:"capacity"`
	Status       string              `json:"status"`
}

// locationClassificationResponse is the resolved subset of a slot's parent
// Zone that a cross-context caller (inventory-storage validating a
// classified SKU's stow target) needs to validate placement, without
// giving that caller the rest of the Zone or LocationSlot shape. This
// context remains the source of truth for these attributes — Zone already
// carries them; this DTO is a denormalized read, not a new aggregate.
type locationClassificationResponse struct {
	Hazmat           bool   `json:"hazmat"`
	TemperatureClass string `json:"temperatureClass"`
}

type importRowResultResponse struct {
	Index        int    `json:"index"`
	LocationCode string `json:"locationCode"`
	Succeeded    bool   `json:"succeeded"`
	Error        string `json:"error,omitempty"`
}

type importReportResponse struct {
	RowsSubmitted int                       `json:"rowsSubmitted"`
	SlotsImported int                       `json:"slotsImported"`
	RowsRejected  int                       `json:"rowsRejected"`
	Results       []importRowResultResponse `json:"results"`
}

// ------------------------------------------------- "draw the warehouse" ----

// siteLayoutResponse is the full nested, drawable structure of one site:
// zones -> aisles -> slots, pre-grouped and pre-ordered so a frontend can
// paint a floor plan without any client-side joining or sorting.
type siteLayoutResponse struct {
	Site   siteResponse             `json:"site"`
	Zones  []zoneLayoutResponse     `json:"zones"`
	Totals siteLayoutTotalsResponse `json:"totals"`
}

type siteLayoutTotalsResponse struct {
	Zones  int `json:"zones"`
	Aisles int `json:"aisles"`
	Slots  int `json:"slots"`
}

type zoneLayoutResponse struct {
	zoneResponse
	Aisles []aisleLayoutResponse `json:"aisles"`
}

type aisleLayoutResponse struct {
	aisleResponse
	Slots []locationSlotResponse `json:"slots"`
}

// zoneGridResponse is one zone's slots as an explicit matrix: rows are
// Levels, columns are (Aisle, Bay) pairs in aisle walk order, and a cell is
// null where the rack has a gap. A UI iterates rows x columns and paints.
type zoneGridResponse struct {
	Zone    zoneResponse         `json:"zone"`
	Columns []gridColumnResponse `json:"columns"`
	Levels  []string             `json:"levels"`
	Rows    []gridRowResponse    `json:"rows"`
}

type gridColumnResponse struct {
	AisleID      string `json:"aisleId"`
	AisleCode    string `json:"aisleCode"`
	Bay          string `json:"bay"`
	SequenceHint int    `json:"sequenceHint"`
}

type gridRowResponse struct {
	Level string `json:"level"`
	// Cells is index-aligned with the grid's Columns. A nil entry is a gap
	// in the rack and serializes as JSON null, per CLAUDE.md.
	Cells []*gridCellResponse `json:"cells"`
}

type gridCellResponse struct {
	Positions []gridPositionResponse `json:"positions"`
}

type gridPositionResponse struct {
	LocationCode string `json:"locationCode"`
	Position     string `json:"position"`
	LocationType string `json:"locationType"`
	Status       string `json:"status"`
}

// ----------------------------------------------------------- RFC 7807 -----

// problemDetails is the RFC 7807 (Problem Details for HTTP APIs) response
// body used for every error response in this service, from day one.
type problemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}
