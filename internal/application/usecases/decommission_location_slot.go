package usecases

import (
	"context"

	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// DecommissionLocationSlot permanently retires a coded slot. One-way: v1
// has no reactivation use case, and re-registering the code is rejected as
// a duplicate rather than quietly resurrecting the slot.
type DecommissionLocationSlot struct {
	Slots  ports.SlotRepo
	Events ports.EventPublisher
	Clock  ports.Clock
}

// Execute decommissions the slot and publishes LocationSlotDecommissioned.
func (uc *DecommissionLocationSlot) Execute(ctx context.Context, code shared.LocationCode) error {
	s, err := uc.Slots.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if s == nil {
		return ErrLocationSlotNotFound
	}
	if err := s.Decommission(); err != nil {
		return err
	}
	if err := uc.Slots.Save(ctx, s); err != nil {
		return err
	}
	return uc.Events.Publish(ctx, shared.NewLocationSlotDecommissioned(uc.Clock.Now(), s.Code()))
}
