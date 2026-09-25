// Package placement holds LocationType (a reusable classification of slot
// shape/kind, with a default capacity envelope) and PlacementRule (which
// LocationTypes are legal in which Zones). Together they are the mechanism
// that stops "ambient product in the frozen zone": the constraint is
// declared once and enforced at slot-registration time, rather than
// re-checked by every caller.
package placement

import "errors"

// ErrUnknownLocationRole is returned when a LocationRole string is not one
// of the known values.
var ErrUnknownLocationRole = errors.New("location role must be one of Storage, Dock, Yard, WorkCenter, Drop, Staging, QC, Consolidation, Shipping")

// LocationRole is what a LocationType is FOR, distinct from its shape/kind.
// Every LocationType in this service meant storage until this role was
// added (ADR-0016): a location can now instead be a dock, a yard slot, a
// work center, a drop zone, a QC bay, a consolidation point, or a shipping
// location — real WMS location roles (Oracle WMS Cloud's Location Master,
// SAP EWM's Storage Type Role), not storage shapes wearing a storage-only
// tag.
type LocationRole string

const (
	// Storage is the default role: the location holds inventory at rest,
	// exactly as every LocationType meant before this role existed.
	Storage LocationRole = "Storage"
	// Dock is used to assign inbound or outbound loads (Oracle: "used to
	// assign to loads during receiving/shipping"; SAP: "Doors").
	Dock LocationRole = "Dock"
	// Yard locates a trailer adjacent to the building (Oracle: "used to
	// locate trailer"; SAP: "Yard").
	Yard LocationRole = "Yard"
	// WorkCenter is a physical area where deconsolidation, inspection,
	// packing, or value-added-service processing takes place (SAP: "Work
	// Center").
	WorkCenter LocationRole = "WorkCenter"
	// Drop holds inbound or outbound LPNs in transit between warehouse
	// processes (Oracle: "used to hold both inbound and outbound LPNs
	// while in between warehouse processes... you can configure the
	// picking task to target a specific drop zone").
	Drop LocationRole = "Drop"
	// RoleStaging is a staging position holding units awaiting the next
	// process step. Named RoleStaging, not Staging, because Staging is
	// already this package's well-known LocationType NAME constant; the
	// role and the name are deliberately independent concepts that happen
	// to share a word.
	RoleStaging LocationRole = "Staging"
	// QC is where goods are checked/inspected, typically during receiving
	// or shipping (Oracle: "QC"; SAP: "Identification Point"/"Pick
	// Point").
	QC LocationRole = "QC"
	// Consolidation groups units bound for the same destination before
	// shipping (Oracle: "used in PTS flow to assign Locations to
	// Destination Stores").
	Consolidation LocationRole = "Consolidation"
	// Shipping is where outbound LPNs are staged for individual dispatch
	// (Oracle: "used for shipping LPNs individually").
	Shipping LocationRole = "Shipping"
)

// ParseLocationRole validates and converts the string form.
func ParseLocationRole(value string) (LocationRole, error) {
	switch LocationRole(value) {
	case Storage, Dock, Yard, WorkCenter, Drop, RoleStaging, QC, Consolidation, Shipping:
		return LocationRole(value), nil
	default:
		return "", ErrUnknownLocationRole
	}
}

// RequiresCapacity reports whether a LocationType with this role must carry
// a strictly positive weight/volume capacity envelope. Storage, staging,
// drop, and consolidation locations hold inventory at rest or in transit,
// exactly like storage always has; a dock door, a yard slot, a work
// center, a QC bay, and a shipping location do not hold a weight/volume of
// stock the way a pallet rack slot does, so capacity is optional for them
// (ADR-0016).
func (r LocationRole) RequiresCapacity() bool {
	switch r {
	case Storage, RoleStaging, Drop, Consolidation:
		return true
	default:
		return false
	}
}
