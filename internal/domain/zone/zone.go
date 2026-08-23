// Package zone holds the Zone aggregate: a behavioral classification scoped
// to a Site. It bundles the Area and Zone segments of a LocationCode into
// one aggregate, because a bare Area carries no rules of its own — every
// PlacementRule is keyed by the behavioral Zone, and a Zone is only
// meaningful inside its Area.
package zone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/claudioed/facility-layout/internal/domain/shared"
)

var (
	// ErrEmptySiteCode is returned when a zone is not scoped to a site.
	ErrEmptySiteCode = errors.New("zone must be scoped to a site code")
	// ErrEmptyAreaCode is returned when the area code is empty.
	ErrEmptyAreaCode = errors.New("area code must not be empty")
	// ErrEmptyZoneCode is returned when the zone code is empty.
	ErrEmptyZoneCode = errors.New("zone code must not be empty")
	// ErrInvalidCode is returned when a site/area/zone code contains a
	// character outside [A-Z0-9].
	ErrInvalidCode = errors.New("zone codes must contain only uppercase letters and digits")
	// ErrAlreadyDecommissioned is returned when decommissioning a zone that
	// is already decommissioned.
	ErrAlreadyDecommissioned = errors.New("zone is already decommissioned")
)

// Zone is a behavioral classification within a Site's area: ambient,
// chilled, frozen, hazmat, forward-pick, reserve. It is not cosmetic — its
// TemperatureClass and Hazmat flag are what PlacementRules match on, and
// its identity is the Site/Area/Zone prefix of every LocationCode inside it.
type Zone struct {
	siteCode         string
	areaCode         string
	zoneCode         string
	temperatureClass shared.TemperatureClass
	hazmat           bool
	status           shared.Status
}

// NewZone validates and constructs an Active Zone scoped to siteCode.
func NewZone(siteCode, areaCode, zoneCode string, temperatureClass shared.TemperatureClass, hazmat bool) (*Zone, error) {
	if siteCode == "" {
		return nil, ErrEmptySiteCode
	}
	if areaCode == "" {
		return nil, ErrEmptyAreaCode
	}
	if zoneCode == "" {
		return nil, ErrEmptyZoneCode
	}
	for _, code := range []string{siteCode, areaCode, zoneCode} {
		if err := validateCode(code); err != nil {
			return nil, err
		}
	}
	if _, err := shared.ParseTemperatureClass(string(temperatureClass)); err != nil {
		return nil, err
	}
	return &Zone{
		siteCode:         siteCode,
		areaCode:         areaCode,
		zoneCode:         zoneCode,
		temperatureClass: temperatureClass,
		hazmat:           hazmat,
		status:           shared.Active,
	}, nil
}

// RehydrateZone rebuilds a Zone from persisted state. Persistence adapters only.
func RehydrateZone(siteCode, areaCode, zoneCode string, temperatureClass shared.TemperatureClass, hazmat bool, status shared.Status) *Zone {
	return &Zone{
		siteCode:         siteCode,
		areaCode:         areaCode,
		zoneCode:         zoneCode,
		temperatureClass: temperatureClass,
		hazmat:           hazmat,
		status:           status,
	}
}

func validateCode(code string) error {
	for _, r := range code {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return fmt.Errorf("%w: %q", ErrInvalidCode, code)
	}
	return nil
}

// ID returns the zone's identity: SITE-AREA-ZONE, e.g. "WH1-STOR-AMB". It
// is exactly the first three segments of every LocationCode inside it.
func (z *Zone) ID() string {
	return strings.Join([]string{z.siteCode, z.areaCode, z.zoneCode}, "-")
}

// SiteCode returns the code of the Site this zone is scoped to.
func (z *Zone) SiteCode() string { return z.siteCode }

// AreaCode returns the coarse functional area segment, e.g. "STOR".
func (z *Zone) AreaCode() string { return z.areaCode }

// ZoneCode returns the behavioral zone segment, e.g. "AMB".
func (z *Zone) ZoneCode() string { return z.zoneCode }

// TemperatureClass returns the zone's thermal class.
func (z *Zone) TemperatureClass() shared.TemperatureClass { return z.temperatureClass }

// Hazmat reports whether the zone is rated for hazardous materials.
func (z *Zone) Hazmat() bool { return z.hazmat }

// Status returns the zone's lifecycle status.
func (z *Zone) Status() shared.Status { return z.status }

// IsActive reports whether new structure may be registered against this zone.
func (z *Zone) IsActive() bool { return z.status == shared.Active }

// Decommission permanently retires the zone. One-way.
func (z *Zone) Decommission() error {
	if z.status == shared.Decommissioned {
		return ErrAlreadyDecommissioned
	}
	z.status = shared.Decommissioned
	return nil
}
