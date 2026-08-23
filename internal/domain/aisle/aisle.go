// Package aisle holds the Aisle aggregate: a physical corridor scoped to a
// Zone. Its SequenceHint is this bounded context's concrete answer to the
// travel-distance input the WES tier currently lacks — it is the aisle's
// walk-order position, which downstream travel-path and congestion
// reasoning reads but never writes.
package aisle

import (
	"errors"
	"fmt"

	"github.com/claudioed/facility-layout/internal/domain/shared"
)

var (
	// ErrEmptyZoneID is returned when an aisle is not scoped to a zone.
	ErrEmptyZoneID = errors.New("aisle must be scoped to a zone id")
	// ErrEmptyAisleCode is returned when the aisle code is empty.
	ErrEmptyAisleCode = errors.New("aisle code must not be empty")
	// ErrInvalidAisleCode is returned when the aisle code contains a
	// character outside [A-Z0-9].
	ErrInvalidAisleCode = errors.New("aisle code must contain only uppercase letters and digits")
	// ErrNegativeSequenceHint is returned when the walk-order hint is negative.
	ErrNegativeSequenceHint = errors.New("aisle sequence hint must not be negative")
	// ErrAlreadyDecommissioned is returned when decommissioning an aisle
	// that is already decommissioned.
	ErrAlreadyDecommissioned = errors.New("aisle is already decommissioned")
)

// Aisle is a physical corridor inside a Zone.
type Aisle struct {
	zoneID       string
	aisleCode    string
	sequenceHint int
	direction    shared.Direction
	status       shared.Status
}

// NewAisle validates and constructs an Active Aisle scoped to zoneID.
func NewAisle(zoneID, aisleCode string, sequenceHint int, direction shared.Direction) (*Aisle, error) {
	if zoneID == "" {
		return nil, ErrEmptyZoneID
	}
	if aisleCode == "" {
		return nil, ErrEmptyAisleCode
	}
	for _, r := range aisleCode {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return nil, fmt.Errorf("%w: %q", ErrInvalidAisleCode, aisleCode)
	}
	if sequenceHint < 0 {
		return nil, ErrNegativeSequenceHint
	}
	if _, err := shared.ParseDirection(string(direction)); err != nil {
		return nil, err
	}
	return &Aisle{
		zoneID:       zoneID,
		aisleCode:    aisleCode,
		sequenceHint: sequenceHint,
		direction:    direction,
		status:       shared.Active,
	}, nil
}

// RehydrateAisle rebuilds an Aisle from persisted state. Persistence adapters only.
func RehydrateAisle(zoneID, aisleCode string, sequenceHint int, direction shared.Direction, status shared.Status) *Aisle {
	return &Aisle{
		zoneID:       zoneID,
		aisleCode:    aisleCode,
		sequenceHint: sequenceHint,
		direction:    direction,
		status:       status,
	}
}

// ID returns the aisle's identity: ZoneID plus the aisle code, e.g.
// "WH1-STOR-AMB-A07" — the first four segments of every LocationCode in it.
func (a *Aisle) ID() string { return a.zoneID + "-" + a.aisleCode }

// ZoneID returns the id of the Zone this aisle is scoped to.
func (a *Aisle) ZoneID() string { return a.zoneID }

// AisleCode returns the aisle segment, e.g. "A07".
func (a *Aisle) AisleCode() string { return a.aisleCode }

// SequenceHint returns the aisle's walk-order position for travel-path
// optimization. Lower comes earlier on the route.
func (a *Aisle) SequenceHint() int { return a.sequenceHint }

// Direction returns whether the aisle is one-way or two-way.
func (a *Aisle) Direction() shared.Direction { return a.direction }

// Status returns the aisle's lifecycle status.
func (a *Aisle) Status() shared.Status { return a.status }

// IsActive reports whether new slots may be registered against this aisle.
func (a *Aisle) IsActive() bool { return a.status == shared.Active }

// Decommission permanently retires the aisle. One-way.
func (a *Aisle) Decommission() error {
	if a.status == shared.Decommissioned {
		return ErrAlreadyDecommissioned
	}
	a.status = shared.Decommissioned
	return nil
}
