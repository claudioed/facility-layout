// Package ports declares the outbound interfaces the application layer
// depends on. Adapters implement these; the application never imports an
// adapter package. This package contains interfaces only.
package ports

import (
	"context"
	"time"

	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/site"
	"github.com/claudioed/facility-layout/internal/domain/slot"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

// SiteRepo persists and retrieves Site aggregates. FindByCode returns
// (nil, nil) when the site does not exist — "not found" is an application
// concern, not a repository error.
type SiteRepo interface {
	Save(ctx context.Context, s *site.Site) error
	FindByCode(ctx context.Context, code string) (*site.Site, error)
	List(ctx context.Context) ([]*site.Site, error)
}

// ZoneRepo persists and retrieves Zone aggregates, keyed by zone id
// (SITE-AREA-ZONE).
type ZoneRepo interface {
	Save(ctx context.Context, z *zone.Zone) error
	FindByID(ctx context.Context, id string) (*zone.Zone, error)
	ListBySite(ctx context.Context, siteCode string) ([]*zone.Zone, error)
}

// AisleRepo persists and retrieves Aisle aggregates, keyed by aisle id
// (SITE-AREA-ZONE-AISLE).
type AisleRepo interface {
	Save(ctx context.Context, a *aisle.Aisle) error
	FindByID(ctx context.Context, id string) (*aisle.Aisle, error)
	ListByZone(ctx context.Context, zoneID string) ([]*aisle.Aisle, error)
}

// SlotRepo persists and retrieves LocationSlot aggregates, keyed by their
// LocationCode (which is their identity).
type SlotRepo interface {
	Save(ctx context.Context, s *slot.LocationSlot) error
	FindByCode(ctx context.Context, code shared.LocationCode) (*slot.LocationSlot, error)
	ListByAisle(ctx context.Context, aisleID string) ([]*slot.LocationSlot, error)
	ListByZone(ctx context.Context, zoneID string) ([]*slot.LocationSlot, error)
}

// LocationTypeRepo persists and retrieves LocationType definitions.
type LocationTypeRepo interface {
	Save(ctx context.Context, t placement.LocationType) error
	FindByName(ctx context.Context, name string) (*placement.LocationType, error)
	List(ctx context.Context) ([]placement.LocationType, error)
}

// PlacementRuleRepo persists and retrieves PlacementRules. List returns the
// full rule set; the use case narrows it to the zone under consideration.
type PlacementRuleRepo interface {
	Save(ctx context.Context, r placement.PlacementRule) error
	FindByID(ctx context.Context, id string) (*placement.PlacementRule, error)
	List(ctx context.Context) ([]placement.PlacementRule, error)
}

// EventPublisher publishes domain events. Adapters may log them, buffer
// them, or forward them to a broker.
type EventPublisher interface {
	Publish(ctx context.Context, event shared.DomainEvent) error
}

// Clock abstracts current time so use cases and tests are deterministic.
type Clock interface {
	Now() time.Time
}

// LocationMetrics records the business-level telemetry this bounded context
// owns: how many coded location slots were registered, and how many were
// turned away and why. Implemented by the telemetry adapter; a use case
// with no recorder wired simply records nothing.
type LocationMetrics interface {
	LocationSlotRegistered(ctx context.Context, outcome string)
}
