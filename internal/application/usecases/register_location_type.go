package usecases

import (
	"context"

	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// RegisterLocationType defines a reusable classification of slot shape/kind
// with a default capacity envelope.
type RegisterLocationType struct {
	LocationTypes ports.LocationTypeRepo
	Events        ports.EventPublisher
	Clock         ports.Clock
}

// Execute registers the location type and publishes LocationTypeRegistered.
func (uc *RegisterLocationType) Execute(ctx context.Context, name string, defaultCapacity shared.Capacity) (placement.LocationType, error) {
	existing, err := uc.LocationTypes.FindByName(ctx, name)
	if err != nil {
		return placement.LocationType{}, err
	}
	if existing != nil {
		return placement.LocationType{}, ErrDuplicateLocationType
	}

	lt, err := placement.NewLocationType(name, defaultCapacity)
	if err != nil {
		return placement.LocationType{}, err
	}
	if err := uc.LocationTypes.Save(ctx, lt); err != nil {
		return placement.LocationType{}, err
	}
	if err := uc.Events.Publish(ctx, shared.NewLocationTypeRegistered(uc.Clock.Now(), lt.Name(), lt.DefaultCapacity())); err != nil {
		return placement.LocationType{}, err
	}
	return lt, nil
}

// ListLocationTypes reads every registered LocationType.
type ListLocationTypes struct {
	LocationTypes ports.LocationTypeRepo
}

// Execute returns every location type.
func (uc *ListLocationTypes) Execute(ctx context.Context) ([]placement.LocationType, error) {
	return uc.LocationTypes.List(ctx)
}
