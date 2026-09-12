// Package aisle holds the Aisle aggregate (a physical corridor scoped to a
// Zone) and CrossAisle (ADR-0017): a zone-scoped connection between two of
// its aisles at a given bay ordinal, letting the travel graph route
// between aisles without walking all the way to either end.
package aisle

import "errors"

var (
	// ErrCrossAisleEmptyZoneID is returned when a cross-aisle is not
	// scoped to a zone.
	ErrCrossAisleEmptyZoneID = errors.New("cross-aisle must be scoped to a zone id")
	// ErrCrossAisleEmptyFromAisle is returned when the from-aisle code is
	// empty.
	ErrCrossAisleEmptyFromAisle = errors.New("cross-aisle requires a from-aisle code")
	// ErrCrossAisleEmptyToAisle is returned when the to-aisle code is
	// empty.
	ErrCrossAisleEmptyToAisle = errors.New("cross-aisle requires a to-aisle code")
	// ErrCrossAisleSameAisle is returned when a cross-aisle names the
	// same aisle at both ends — a connection from an aisle to itself
	// carries no travel-distance information.
	ErrCrossAisleSameAisle = errors.New("cross-aisle must connect two distinct aisles")
	// ErrCrossAisleEmptyBay is returned when the bay ordinal is empty.
	ErrCrossAisleEmptyBay = errors.New("cross-aisle requires a bay")
	// ErrCrossAisleAlreadyDecommissioned is returned when decommissioning
	// a cross-aisle that is already decommissioned.
	ErrCrossAisleAlreadyDecommissioned = errors.New("cross-aisle is already decommissioned")
)

// crossAisleStatus mirrors shared.Status's Active/Decommissioned values.
// CrossAisle uses its own tiny status rather than importing shared.Status
// so this file has no cross-package status-widening surface; only Active
// and Decommissioned are meaningful for a connection (there is no
// "under maintenance" state for a walkway).
type crossAisleStatus string

const (
	crossAisleActive         crossAisleStatus = "Active"
	crossAisleDecommissioned crossAisleStatus = "Decommissioned"
)

// CrossAisle is a zone-scoped connection between two of its aisles at a
// given bay: physically, a walkway a picker can cross between FromAisle
// and ToAisle without walking to either aisle's end. It carries no
// geometry of its own — the travel graph derives the connection's length
// from the two aisles' centrelines at AtBay when both are known, or from
// the zone's aisle pitch otherwise (ADR-0017).
type CrossAisle struct {
	zoneID    string
	fromAisle string
	toAisle   string
	atBay     string
	status    crossAisleStatus
}

// NewCrossAisle validates and constructs an Active CrossAisle. fromAisle
// and toAisle are aisle codes (not full aisle ids) scoped to zoneID; the
// use case is responsible for checking both actually exist in that zone
// before calling this constructor — CrossAisle itself only enforces the
// shape invariants it can check without a repository.
func NewCrossAisle(zoneID, fromAisle, toAisle, atBay string) (*CrossAisle, error) {
	if zoneID == "" {
		return nil, ErrCrossAisleEmptyZoneID
	}
	if fromAisle == "" {
		return nil, ErrCrossAisleEmptyFromAisle
	}
	if toAisle == "" {
		return nil, ErrCrossAisleEmptyToAisle
	}
	if fromAisle == toAisle {
		return nil, ErrCrossAisleSameAisle
	}
	if atBay == "" {
		return nil, ErrCrossAisleEmptyBay
	}
	return &CrossAisle{zoneID: zoneID, fromAisle: fromAisle, toAisle: toAisle, atBay: atBay, status: crossAisleActive}, nil
}

// RehydrateCrossAisle rebuilds a CrossAisle from persisted state.
// Persistence adapters only.
func RehydrateCrossAisle(zoneID, fromAisle, toAisle, atBay string, decommissioned bool) *CrossAisle {
	status := crossAisleActive
	if decommissioned {
		status = crossAisleDecommissioned
	}
	return &CrossAisle{zoneID: zoneID, fromAisle: fromAisle, toAisle: toAisle, atBay: atBay, status: status}
}

// ZoneID returns the id of the Zone this cross-aisle is scoped to.
func (c *CrossAisle) ZoneID() string { return c.zoneID }

// FromAisle returns one end's aisle code.
func (c *CrossAisle) FromAisle() string { return c.fromAisle }

// ToAisle returns the other end's aisle code.
func (c *CrossAisle) ToAisle() string { return c.toAisle }

// AtBay returns the bay ordinal the connection is made at.
func (c *CrossAisle) AtBay() string { return c.atBay }

// IsActive reports whether the travel graph should treat this connection
// as usable.
func (c *CrossAisle) IsActive() bool { return c.status == crossAisleActive }

// Decommission permanently retires the cross-aisle (e.g. a walkway was
// physically closed off). One-way, like every other lifecycle transition
// in this context.
func (c *CrossAisle) Decommission() error {
	if c.status == crossAisleDecommissioned {
		return ErrCrossAisleAlreadyDecommissioned
	}
	c.status = crossAisleDecommissioned
	return nil
}

// Connects reports whether this cross-aisle links aisleCode to the other
// aisle in the pair, returning that other aisle's code. ok is false when
// aisleCode is neither end.
func (c *CrossAisle) Connects(aisleCode string) (other string, ok bool) {
	switch aisleCode {
	case c.fromAisle:
		return c.toAisle, true
	case c.toAisle:
		return c.fromAisle, true
	default:
		return "", false
	}
}
