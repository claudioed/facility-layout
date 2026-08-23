// Package site holds the Site aggregate: a physical facility/building and
// the root of the location hierarchy. Everything else in this bounded
// context hangs off a Site.
package site

import (
	"errors"
	"fmt"

	"github.com/claudioed/facility-layout/internal/domain/shared"
)

var (
	// ErrEmptySiteCode is returned when a site code is empty.
	ErrEmptySiteCode = errors.New("site code must not be empty")
	// ErrInvalidSiteCode is returned when a site code contains a character
	// outside [A-Z0-9].
	ErrInvalidSiteCode = errors.New("site code must contain only uppercase letters and digits")
	// ErrEmptySiteName is returned when a site's human name is empty.
	ErrEmptySiteName = errors.New("site name must not be empty")
	// ErrAlreadyDecommissioned is returned when decommissioning a site that
	// is already decommissioned.
	ErrAlreadyDecommissioned = errors.New("site is already decommissioned")
)

// Site is a physical facility/building. Its identity is its SiteCode, which
// is also the first segment of every LocationCode inside it. Uniqueness of
// the code is enforced at the application/repository layer: a single
// aggregate cannot see its siblings.
type Site struct {
	code   string
	name   string
	status shared.Status
}

// NewSite validates and constructs an Active Site.
func NewSite(code, name string) (*Site, error) {
	if err := validateSiteCode(code); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, ErrEmptySiteName
	}
	return &Site{code: code, name: name, status: shared.Active}, nil
}

// RehydrateSite rebuilds a Site from persisted state without re-running the
// registration invariants. Only persistence adapters may call it.
func RehydrateSite(code, name string, status shared.Status) *Site {
	return &Site{code: code, name: name, status: status}
}

func validateSiteCode(code string) error {
	if code == "" {
		return ErrEmptySiteCode
	}
	for _, r := range code {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return fmt.Errorf("%w: %q", ErrInvalidSiteCode, code)
	}
	return nil
}

// Code returns the site code, e.g. "WH1".
func (s *Site) Code() string { return s.code }

// Name returns the human-readable site name.
func (s *Site) Name() string { return s.name }

// Status returns the site's lifecycle status.
func (s *Site) Status() shared.Status { return s.status }

// IsActive reports whether new structure may be registered against this site.
func (s *Site) IsActive() bool { return s.status == shared.Active }

// Decommission permanently retires the site. One-way: a decommissioned site
// is never reactivated.
func (s *Site) Decommission() error {
	if s.status == shared.Decommissioned {
		return ErrAlreadyDecommissioned
	}
	s.status = shared.Decommissioned
	return nil
}
