package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ScopeName identifies this service's own (non-library) instrumentation in
// the exported telemetry.
const ScopeName = "github.com/claudioed/facility-layout"

// LocationMetrics is the OTel-backed implementation of the application's
// ports.LocationMetrics: it counts every attempt to register a coded
// location slot, split by outcome, so "how many slots were rejected by a
// placement rule today?" is a query rather than a log grep.
//
// The counter is the business signal this service exists to produce; it is
// recorded in the use case, not in an HTTP handler, so a slot registered by
// a bulk import counts exactly the same as one registered by a POST.
type LocationMetrics struct {
	registrations metric.Int64Counter
}

// NewLocationMetrics builds the recorder against the global MeterProvider,
// which Setup installs. Calling it before Setup is harmless: the global
// provider delegates, so instruments created early start recording once a
// real provider is in place.
func NewLocationMetrics() (*LocationMetrics, error) {
	meter := otel.GetMeterProvider().Meter(ScopeName)

	registrations, err := meter.Int64Counter(
		"facility.location_slot.registrations",
		metric.WithDescription("Attempts to register a coded location slot, by outcome."),
		metric.WithUnit("{registration}"),
	)
	if err != nil {
		return nil, err
	}

	return &LocationMetrics{registrations: registrations}, nil
}

// LocationSlotRegistered records one registration attempt. outcome is one
// of the application layer's registration outcomes (accepted,
// rejected_by_placement_rule, rejected).
func (m *LocationMetrics) LocationSlotRegistered(ctx context.Context, outcome string) {
	m.registrations.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
}
