package http

import (
	"errors"
	"net/http"

	"github.com/claudioed/facility-layout/internal/application/usecases"
	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/site"
	"github.com/claudioed/facility-layout/internal/domain/slot"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

// statusFor maps a typed domain/application error to an HTTP status code.
//
//   - 404: the named site/zone/aisle/slot/type/rule does not exist.
//   - 409: a genuine state conflict — a code already taken, a parent that
//     exists but is no longer Active, a slot already decommissioned.
//   - 422: syntactically fine but semantically invalid — a non-positive
//     capacity, an unknown enum value, a PlacementRule violation.
//   - 400: malformed input — a location code that is not seven [A-Z0-9]
//     segments, a missing required field, a body that is not JSON.
func statusFor(err error) int {
	switch {
	case errors.Is(err, usecases.ErrSiteNotFound),
		errors.Is(err, usecases.ErrZoneNotFound),
		errors.Is(err, usecases.ErrAisleNotFound),
		errors.Is(err, usecases.ErrLocationSlotNotFound),
		errors.Is(err, usecases.ErrLocationTypeNotFound),
		errors.Is(err, usecases.ErrPlacementRuleNotFound):
		return http.StatusNotFound

	case errors.Is(err, usecases.ErrDuplicateSite),
		errors.Is(err, usecases.ErrDuplicateZone),
		errors.Is(err, usecases.ErrDuplicateAisle),
		errors.Is(err, usecases.ErrDuplicateLocationType),
		errors.Is(err, usecases.ErrDuplicatePlacementRule),
		errors.Is(err, usecases.ErrDuplicateLocationCode),
		errors.Is(err, usecases.ErrSiteNotActive),
		errors.Is(err, usecases.ErrZoneNotActive),
		errors.Is(err, usecases.ErrAisleNotActive),
		errors.Is(err, site.ErrAlreadyDecommissioned),
		errors.Is(err, zone.ErrAlreadyDecommissioned),
		errors.Is(err, aisle.ErrAlreadyDecommissioned),
		errors.Is(err, slot.ErrAlreadyDecommissioned):
		return http.StatusConflict

	case errors.Is(err, placement.ErrPlacementRuleViolated),
		errors.Is(err, shared.ErrInvalidMaxWeight),
		errors.Is(err, shared.ErrInvalidMaxVolume),
		errors.Is(err, shared.ErrUnknownTemperatureClass),
		errors.Is(err, shared.ErrUnknownDirection),
		errors.Is(err, shared.ErrUnknownStatus),
		errors.Is(err, placement.ErrUnknownEffect),
		errors.Is(err, placement.ErrEmptyPredicate),
		errors.Is(err, aisle.ErrNegativeSequenceHint),
		errors.Is(err, slot.ErrZoneMismatch):
		return http.StatusUnprocessableEntity

	case errors.Is(err, shared.ErrMalformedLocationCode),
		errors.Is(err, shared.ErrEmptyLocationSegment),
		errors.Is(err, shared.ErrInvalidLocationSegment),
		errors.Is(err, site.ErrEmptySiteCode),
		errors.Is(err, site.ErrInvalidSiteCode),
		errors.Is(err, site.ErrEmptySiteName),
		errors.Is(err, zone.ErrEmptySiteCode),
		errors.Is(err, zone.ErrEmptyAreaCode),
		errors.Is(err, zone.ErrEmptyZoneCode),
		errors.Is(err, zone.ErrInvalidCode),
		errors.Is(err, aisle.ErrEmptyZoneID),
		errors.Is(err, aisle.ErrEmptyAisleCode),
		errors.Is(err, aisle.ErrInvalidAisleCode),
		errors.Is(err, placement.ErrEmptyLocationTypeName),
		errors.Is(err, placement.ErrEmptyRuleID),
		errors.Is(err, placement.ErrEmptyRuleLocationType),
		errors.Is(err, slot.ErrMissingLocationCode),
		errors.Is(err, slot.ErrMissingLocationType),
		errors.Is(err, usecases.ErrEmptyImport):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

// problemBaseURI is the namespace for this service's RFC 7807 "type" URIs.
// It does not need to resolve to a real page — it's an identifier, unique
// per distinct error category in this service.
const problemBaseURI = "https://errors.facility-layout.warehouse-systems.dev/"

// problemInfo is the fixed, category-level (type, title) pair for an RFC
// 7807 problem response. slug becomes the last path segment of "type";
// title is a fixed human string for the category (the dynamic detail comes
// from err.Error() at write time, not from this table).
type problemInfo struct {
	slug  string
	title string
}

// problemFor maps a typed domain/application error to its RFC 7807
// (type, title) pair, mirroring statusFor's groupings one-for-one.
func problemFor(err error) problemInfo {
	switch {
	case errors.Is(err, usecases.ErrSiteNotFound):
		return problemInfo{"site-not-found", "Site not found"}
	case errors.Is(err, usecases.ErrZoneNotFound):
		return problemInfo{"zone-not-found", "Zone not found"}
	case errors.Is(err, usecases.ErrAisleNotFound):
		return problemInfo{"aisle-not-found", "Aisle not found"}
	case errors.Is(err, usecases.ErrLocationSlotNotFound):
		return problemInfo{"location-slot-not-found", "Location slot not found"}
	case errors.Is(err, usecases.ErrLocationTypeNotFound):
		return problemInfo{"location-type-not-found", "Location type not found"}
	case errors.Is(err, usecases.ErrPlacementRuleNotFound):
		return problemInfo{"placement-rule-not-found", "Placement rule not found"}

	case errors.Is(err, usecases.ErrDuplicateSite):
		return problemInfo{"duplicate-site-code", "A site with this code already exists"}
	case errors.Is(err, usecases.ErrDuplicateZone):
		return problemInfo{"duplicate-zone", "A zone with this area and zone code already exists in this site"}
	case errors.Is(err, usecases.ErrDuplicateAisle):
		return problemInfo{"duplicate-aisle", "An aisle with this code already exists in this zone"}
	case errors.Is(err, usecases.ErrDuplicateLocationType):
		return problemInfo{"duplicate-location-type", "A location type with this name already exists"}
	case errors.Is(err, usecases.ErrDuplicatePlacementRule):
		return problemInfo{"duplicate-placement-rule", "A placement rule with this id already exists"}
	case errors.Is(err, usecases.ErrDuplicateLocationCode):
		return problemInfo{"duplicate-location-code", "A location slot with this code already exists"}

	case errors.Is(err, usecases.ErrSiteNotActive):
		return problemInfo{"site-not-active", "Site is not active"}
	case errors.Is(err, usecases.ErrZoneNotActive):
		return problemInfo{"zone-not-active", "Zone is not active"}
	case errors.Is(err, usecases.ErrAisleNotActive):
		return problemInfo{"aisle-not-active", "Aisle is not active"}

	case errors.Is(err, site.ErrAlreadyDecommissioned),
		errors.Is(err, zone.ErrAlreadyDecommissioned),
		errors.Is(err, aisle.ErrAlreadyDecommissioned),
		errors.Is(err, slot.ErrAlreadyDecommissioned):
		return problemInfo{"already-decommissioned", "This structure is already decommissioned"}

	case errors.Is(err, placement.ErrPlacementRuleViolated):
		return problemInfo{"placement-rule-violated", "Location type is not legal in this zone"}
	case errors.Is(err, shared.ErrInvalidMaxWeight):
		return problemInfo{"invalid-max-weight", "Capacity max weight must be greater than zero"}
	case errors.Is(err, shared.ErrInvalidMaxVolume):
		return problemInfo{"invalid-max-volume", "Capacity max volume must be greater than zero"}
	case errors.Is(err, shared.ErrUnknownTemperatureClass):
		return problemInfo{"unknown-temperature-class", "Unknown temperature class"}
	case errors.Is(err, shared.ErrUnknownDirection):
		return problemInfo{"unknown-direction", "Unknown aisle direction"}
	case errors.Is(err, shared.ErrUnknownStatus):
		return problemInfo{"unknown-status", "Unknown lifecycle status"}
	case errors.Is(err, placement.ErrUnknownEffect):
		return problemInfo{"unknown-placement-effect", "Unknown placement rule effect"}
	case errors.Is(err, placement.ErrEmptyPredicate):
		return problemInfo{"empty-zone-predicate", "Placement rule predicate constrains nothing"}
	case errors.Is(err, aisle.ErrNegativeSequenceHint):
		return problemInfo{"negative-sequence-hint", "Aisle sequence hint must not be negative"}
	case errors.Is(err, slot.ErrZoneMismatch):
		return problemInfo{"zone-mismatch", "Zone attributes do not match the location code's zone"}

	case errors.Is(err, shared.ErrMalformedLocationCode),
		errors.Is(err, shared.ErrEmptyLocationSegment),
		errors.Is(err, shared.ErrInvalidLocationSegment):
		return problemInfo{"malformed-location-code", "Malformed location code"}
	case errors.Is(err, site.ErrEmptySiteCode), errors.Is(err, site.ErrInvalidSiteCode), errors.Is(err, zone.ErrEmptySiteCode):
		return problemInfo{"invalid-site-code", "Invalid site code"}
	case errors.Is(err, site.ErrEmptySiteName):
		return problemInfo{"empty-site-name", "Site name must not be empty"}
	case errors.Is(err, zone.ErrEmptyAreaCode), errors.Is(err, zone.ErrEmptyZoneCode), errors.Is(err, zone.ErrInvalidCode):
		return problemInfo{"invalid-zone-code", "Invalid area or zone code"}
	case errors.Is(err, aisle.ErrEmptyZoneID), errors.Is(err, aisle.ErrEmptyAisleCode), errors.Is(err, aisle.ErrInvalidAisleCode):
		return problemInfo{"invalid-aisle-code", "Invalid aisle code"}
	case errors.Is(err, placement.ErrEmptyLocationTypeName), errors.Is(err, slot.ErrMissingLocationType), errors.Is(err, placement.ErrEmptyRuleLocationType):
		return problemInfo{"invalid-location-type", "Invalid location type"}
	case errors.Is(err, placement.ErrEmptyRuleID):
		return problemInfo{"empty-placement-rule-id", "Placement rule id must not be empty"}
	case errors.Is(err, slot.ErrMissingLocationCode):
		return problemInfo{"missing-location-code", "Location code is required"}
	case errors.Is(err, usecases.ErrEmptyImport):
		return problemInfo{"empty-import", "Facility layout import must contain at least one row"}

	default:
		return problemInfo{"internal-error", "An unexpected internal error occurred"}
	}
}
