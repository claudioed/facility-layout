package shared

import "time"

// eventTypePrefix is this bounded context's CloudEvents `type` namespace.
// The convention is identical to the other four warehouse-systems services:
//
//	com.warehouse.<subdomain>.<bounded-context>.<entity>.<EventName>
//
// reverse-DNS, lowercase except the final PascalCase event name, and the
// entity segment carries no hyphen even for multi-word aggregate names.
// This service's subdomain segment is `wms`: bin-accurate location is
// WMS-tier in the domain reference, and this service is the generalized,
// multi-consumer version of that concern.
const eventTypePrefix = "com.warehouse.wms.facility-layout."

// DomainEvent is a past-tense fact published by an aggregate. Adapters
// (outbound/events) serialize and publish these; the domain never depends
// on the publishing mechanism. EventType is this context's Published
// Language: downstream Conformists key off it.
type DomainEvent interface {
	EventName() string
	EventType() string
	OccurredAt() time.Time
}

type base struct {
	Name string    `json:"eventName"`
	Type string    `json:"eventType"`
	At   time.Time `json:"occurredAt"`
}

func (b base) EventName() string     { return b.Name }
func (b base) EventType() string     { return b.Type }
func (b base) OccurredAt() time.Time { return b.At }

func newBase(entity, name string, occurredAt time.Time) base {
	return base{Name: name, Type: eventTypePrefix + entity + "." + name, At: occurredAt}
}

// SiteRegistered: a physical facility was added to the warehouse map.
type SiteRegistered struct {
	base
	SiteCode string `json:"siteCode"`
	SiteName string `json:"siteName"`
}

// NewSiteRegistered builds a SiteRegistered event.
func NewSiteRegistered(occurredAt time.Time, siteCode, siteName string) SiteRegistered {
	return SiteRegistered{base: newBase("site", "SiteRegistered", occurredAt), SiteCode: siteCode, SiteName: siteName}
}

// ZoneRegistered: a behavioral zone was added inside a Site's area.
type ZoneRegistered struct {
	base
	ZoneID           string           `json:"zoneId"`
	SiteCode         string           `json:"siteCode"`
	AreaCode         string           `json:"areaCode"`
	ZoneCode         string           `json:"zoneCode"`
	TemperatureClass TemperatureClass `json:"temperatureClass"`
	Hazmat           bool             `json:"hazmat"`
}

// NewZoneRegistered builds a ZoneRegistered event.
func NewZoneRegistered(occurredAt time.Time, zoneID, siteCode, areaCode, zoneCode string, temperatureClass TemperatureClass, hazmat bool) ZoneRegistered {
	return ZoneRegistered{
		base:             newBase("zone", "ZoneRegistered", occurredAt),
		ZoneID:           zoneID,
		SiteCode:         siteCode,
		AreaCode:         areaCode,
		ZoneCode:         zoneCode,
		TemperatureClass: temperatureClass,
		Hazmat:           hazmat,
	}
}

// AisleRegistered: a physical corridor was added inside a Zone.
type AisleRegistered struct {
	base
	AisleID      string    `json:"aisleId"`
	ZoneID       string    `json:"zoneId"`
	AisleCode    string    `json:"aisleCode"`
	SequenceHint int       `json:"sequenceHint"`
	Direction    Direction `json:"direction"`
}

// NewAisleRegistered builds an AisleRegistered event.
func NewAisleRegistered(occurredAt time.Time, aisleID, zoneID, aisleCode string, sequenceHint int, direction Direction) AisleRegistered {
	return AisleRegistered{
		base:         newBase("aisle", "AisleRegistered", occurredAt),
		AisleID:      aisleID,
		ZoneID:       zoneID,
		AisleCode:    aisleCode,
		SequenceHint: sequenceHint,
		Direction:    direction,
	}
}

// LocationTypeRegistered: a reusable slot shape/kind was defined.
type LocationTypeRegistered struct {
	base
	LocationType string  `json:"locationType"`
	MaxWeightKg  float64 `json:"maxWeightKg"`
	MaxVolumeM3  float64 `json:"maxVolumeM3"`
}

// NewLocationTypeRegistered builds a LocationTypeRegistered event.
func NewLocationTypeRegistered(occurredAt time.Time, locationType string, capacity Capacity) LocationTypeRegistered {
	return LocationTypeRegistered{
		base:         newBase("locationtype", "LocationTypeRegistered", occurredAt),
		LocationType: locationType,
		MaxWeightKg:  capacity.MaxWeightKg(),
		MaxVolumeM3:  capacity.MaxVolumeM3(),
	}
}

// PlacementRuleDefined: a rule constraining which LocationTypes are legal
// in which Zones was declared.
type PlacementRuleDefined struct {
	base
	RuleID       string `json:"ruleId"`
	LocationType string `json:"locationType"`
	Effect       string `json:"effect"`
	Predicate    string `json:"predicate"`
}

// NewPlacementRuleDefined builds a PlacementRuleDefined event.
func NewPlacementRuleDefined(occurredAt time.Time, ruleID, locationType, effect, predicate string) PlacementRuleDefined {
	return PlacementRuleDefined{
		base:         newBase("placementrule", "PlacementRuleDefined", occurredAt),
		RuleID:       ruleID,
		LocationType: locationType,
		Effect:       effect,
		Predicate:    predicate,
	}
}

// LocationSlotRegistered: a coded leaf slot now exists on the warehouse map.
type LocationSlotRegistered struct {
	base
	LocationCode string  `json:"locationCode"`
	AisleID      string  `json:"aisleId"`
	ZoneID       string  `json:"zoneId"`
	LocationType string  `json:"locationType"`
	MaxWeightKg  float64 `json:"maxWeightKg"`
	MaxVolumeM3  float64 `json:"maxVolumeM3"`
}

// NewLocationSlotRegistered builds a LocationSlotRegistered event.
func NewLocationSlotRegistered(occurredAt time.Time, code LocationCode, locationType string, capacity Capacity) LocationSlotRegistered {
	return LocationSlotRegistered{
		base:         newBase("locationslot", "LocationSlotRegistered", occurredAt),
		LocationCode: code.String(),
		AisleID:      code.AisleID(),
		ZoneID:       code.ZoneID(),
		LocationType: locationType,
		MaxWeightKg:  capacity.MaxWeightKg(),
		MaxVolumeM3:  capacity.MaxVolumeM3(),
	}
}

// LocationSlotDecommissioned: a coded slot was permanently retired.
type LocationSlotDecommissioned struct {
	base
	LocationCode string `json:"locationCode"`
}

// NewLocationSlotDecommissioned builds a LocationSlotDecommissioned event.
func NewLocationSlotDecommissioned(occurredAt time.Time, code LocationCode) LocationSlotDecommissioned {
	return LocationSlotDecommissioned{
		base:         newBase("locationslot", "LocationSlotDecommissioned", occurredAt),
		LocationCode: code.String(),
	}
}

// FacilityLayoutImported: a bulk layout import completed. Emitted once per
// import call; the individual LocationSlotRegistered events still fire
// per-slot within that same import.
type FacilityLayoutImported struct {
	base
	RowsSubmitted int `json:"rowsSubmitted"`
	SlotsImported int `json:"slotsImported"`
	RowsRejected  int `json:"rowsRejected"`
}

// NewFacilityLayoutImported builds a FacilityLayoutImported event.
func NewFacilityLayoutImported(occurredAt time.Time, submitted, imported, rejected int) FacilityLayoutImported {
	return FacilityLayoutImported{
		base:          newBase("locationslot", "FacilityLayoutImported", occurredAt),
		RowsSubmitted: submitted,
		SlotsImported: imported,
		RowsRejected:  rejected,
	}
}
