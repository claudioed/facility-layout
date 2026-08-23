// Package events provides outbound EventPublisher implementations. The
// interface is intentionally the shape a Kafka producer would satisfy
// (Publish(ctx, event) error), so a broker-backed publisher can be dropped
// in later without touching the application layer — this service's domain
// events are its Published Language, and every event already carries its
// CloudEvents `type`.
package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// LogPublisher publishes domain events by logging them as JSON. Useful for
// local development and as a default when no broker is configured.
type LogPublisher struct {
	logger *slog.Logger
}

// NewLogPublisher builds a LogPublisher writing to logger.
func NewLogPublisher(logger *slog.Logger) *LogPublisher {
	return &LogPublisher{logger: logger}
}

// Publish logs the event as JSON, tagged with its CloudEvents type.
func (p *LogPublisher) Publish(_ context.Context, event shared.DomainEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	p.logger.Info("domain event published",
		"event_name", event.EventName(),
		"event_type", event.EventType(),
		"payload", json.RawMessage(payload),
	)
	return nil
}
