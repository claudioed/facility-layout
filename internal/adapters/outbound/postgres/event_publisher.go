package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// EventPublisher is a pgxpool-backed implementation of
// ports.EventPublisher. It appends every domain event to an `events` table,
// giving this service a durable outbox that a broker relay can later drain
// without the application layer knowing about it.
type EventPublisher struct {
	pool *pgxpool.Pool
}

// NewEventPublisher builds an EventPublisher over pool.
func NewEventPublisher(pool *pgxpool.Pool) *EventPublisher {
	return &EventPublisher{pool: pool}
}

// Publish appends the event to the events table.
func (p *EventPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx, `
		INSERT INTO events (event_name, event_type, occurred_at, payload) VALUES ($1, $2, $3, $4)
	`, event.EventName(), event.EventType(), event.OccurredAt(), payload)
	return err
}
