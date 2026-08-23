package shared

import (
	"fmt"
	"strings"
)

// segmentCount is the number of segments in a location code: Site, Area,
// Zone, Aisle, Bay, Level, Position.
const segmentCount = 7

// LocationCode is the industry-standard, human-parsable coded address of a
// physical slot, built from seven typed segments and rendered
// hyphen-joined, coarsest to finest:
//
//	WH1-STOR-AMB-A07-03-02-B
//	 |    |    |   |   |  |  `-- Position: left-to-right slot on the level
//	 |    |    |   |   |  `----- Level:    vertical level/shelf
//	 |    |    |   |   `-------- Bay:      bay/section along the aisle
//	 |    |    |   `------------ Aisle:    physical corridor
//	 |    |    `---------------- Zone:     behavioral class (AMB/CHL/FRZ/HAZ/...)
//	 |    `--------------------- Area:     coarse functional area (STOR/RCV/...)
//	 `-------------------------- Site:     the physical facility
//
// It is a value object, never a free-text string: it always round-trips
// through String()/ParseLocationCode(), and construction is rejected if any
// segment is empty or contains a character other than [A-Z0-9].
type LocationCode struct {
	site     string
	area     string
	zone     string
	aisle    string
	bay      string
	level    string
	position string
}

// NewLocationCode validates the seven segments and builds a LocationCode.
func NewLocationCode(site, area, zone, aisle, bay, level, position string) (LocationCode, error) {
	segments := [segmentCount]string{site, area, zone, aisle, bay, level, position}
	names := [segmentCount]string{"site", "area", "zone", "aisle", "bay", "level", "position"}

	for i, segment := range segments {
		if err := validateSegment(names[i], segment); err != nil {
			return LocationCode{}, err
		}
	}

	return LocationCode{
		site:     site,
		area:     area,
		zone:     zone,
		aisle:    aisle,
		bay:      bay,
		level:    level,
		position: position,
	}, nil
}

// ParseLocationCode parses the hyphen-joined string form produced by
// String() back into a LocationCode, applying the same validation.
func ParseLocationCode(value string) (LocationCode, error) {
	parts := strings.Split(value, "-")
	if len(parts) != segmentCount {
		return LocationCode{}, fmt.Errorf("%w: got %d segments in %q", ErrMalformedLocationCode, len(parts), value)
	}
	return NewLocationCode(parts[0], parts[1], parts[2], parts[3], parts[4], parts[5], parts[6])
}

func validateSegment(name, value string) error {
	if value == "" {
		return fmt.Errorf("%w: %s", ErrEmptyLocationSegment, name)
	}
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return fmt.Errorf("%w: %s segment %q", ErrInvalidLocationSegment, name, value)
	}
	return nil
}

// String renders the canonical hyphen-joined form, e.g.
// "WH1-STOR-AMB-A07-03-02-B".
func (c LocationCode) String() string {
	return strings.Join([]string{c.site, c.area, c.zone, c.aisle, c.bay, c.level, c.position}, "-")
}

// Site returns the site segment.
func (c LocationCode) Site() string { return c.site }

// Area returns the area segment.
func (c LocationCode) Area() string { return c.area }

// Zone returns the zone segment.
func (c LocationCode) Zone() string { return c.zone }

// Aisle returns the aisle segment.
func (c LocationCode) Aisle() string { return c.aisle }

// Bay returns the bay segment.
func (c LocationCode) Bay() string { return c.bay }

// Level returns the level segment.
func (c LocationCode) Level() string { return c.level }

// Position returns the position segment.
func (c LocationCode) Position() string { return c.position }

// ZoneID returns the identity of the Zone aggregate this code resolves to:
// the Site, Area and Zone segments joined, e.g. "WH1-STOR-AMB".
func (c LocationCode) ZoneID() string {
	return strings.Join([]string{c.site, c.area, c.zone}, "-")
}

// AisleID returns the identity of the Aisle aggregate this code resolves
// to: ZoneID plus the aisle segment, e.g. "WH1-STOR-AMB-A07".
func (c LocationCode) AisleID() string {
	return c.ZoneID() + "-" + c.aisle
}

// IsZero reports whether this is the zero LocationCode (never produced by a
// successful construction, since every segment must be non-empty).
func (c LocationCode) IsZero() bool {
	return c == LocationCode{}
}
