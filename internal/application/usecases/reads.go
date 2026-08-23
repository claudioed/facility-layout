package usecases

import (
	"context"

	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/domain/aisle"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/zone"
)

// The single-resource reads below exist so that every 201 this service
// returns can carry a Location header that actually resolves: a Location
// pointing at a URL with no GET is not REST maturity level 2.

// GetZone reads one Zone by its id.
type GetZone struct {
	Zones ports.ZoneRepo
}

// Execute returns the zone, or ErrZoneNotFound.
func (uc *GetZone) Execute(ctx context.Context, id string) (*zone.Zone, error) {
	z, err := uc.Zones.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, ErrZoneNotFound
	}
	return z, nil
}

// GetAisle reads one Aisle by its id.
type GetAisle struct {
	Aisles ports.AisleRepo
}

// Execute returns the aisle, or ErrAisleNotFound.
func (uc *GetAisle) Execute(ctx context.Context, id string) (*aisle.Aisle, error) {
	a, err := uc.Aisles.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAisleNotFound
	}
	return a, nil
}

// GetLocationType reads one LocationType by its name.
type GetLocationType struct {
	LocationTypes ports.LocationTypeRepo
}

// Execute returns the location type, or ErrLocationTypeNotFound.
func (uc *GetLocationType) Execute(ctx context.Context, name string) (placement.LocationType, error) {
	t, err := uc.LocationTypes.FindByName(ctx, name)
	if err != nil {
		return placement.LocationType{}, err
	}
	if t == nil {
		return placement.LocationType{}, ErrLocationTypeNotFound
	}
	return *t, nil
}

// GetPlacementRule reads one PlacementRule by its id.
type GetPlacementRule struct {
	Rules ports.PlacementRuleRepo
}

// Execute returns the rule, or ErrPlacementRuleNotFound.
func (uc *GetPlacementRule) Execute(ctx context.Context, id string) (placement.PlacementRule, error) {
	rule, err := uc.Rules.FindByID(ctx, id)
	if err != nil {
		return placement.PlacementRule{}, err
	}
	if rule == nil {
		return placement.PlacementRule{}, ErrPlacementRuleNotFound
	}
	return *rule, nil
}
