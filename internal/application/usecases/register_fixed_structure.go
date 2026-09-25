package usecases

import (
	"context"

	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/domain/shared"
	"github.com/claudioed/facility-layout/internal/domain/structure"
)

// RegisterFixedStructure adds a site-scoped physical obstacle (wall,
// column, office, conveyor, or other) to the warehouse map (ADR-0017). A
// structure cannot be registered against an unknown site. id is the
// structure's opaque identity, supplied by the caller — the same pattern
// PlacementRule ids follow (the inbound adapter mints a UUID when the
// caller does not name its own id).
type RegisterFixedStructure struct {
	Sites      ports.SiteRepo
	Structures ports.FixedStructureRepo
	Events     ports.EventPublisher
	Clock      ports.Clock
}

// Execute registers the structure and publishes FixedStructureRegistered.
func (uc *RegisterFixedStructure) Execute(ctx context.Context, id, siteCode string, kind structure.Kind, footprint shared.Rect, label string) (*structure.FixedStructure, error) {
	site, err := uc.Sites.FindByCode(ctx, siteCode)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, ErrSiteNotFound
	}

	existing, err := uc.Structures.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateFixedStructure
	}

	f, err := structure.NewFixedStructure(id, siteCode, kind, footprint, label)
	if err != nil {
		return nil, err
	}
	if err := uc.Structures.Save(ctx, f); err != nil {
		return nil, err
	}
	event := shared.NewFixedStructureRegistered(uc.Clock.Now(), f.ID(), f.SiteCode(), string(f.Kind()), f.Footprint(), f.Label())
	if err := uc.Events.Publish(ctx, event); err != nil {
		return nil, err
	}
	return f, nil
}
