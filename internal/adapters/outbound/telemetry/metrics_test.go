package telemetry_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/claudioed/facility-layout/internal/adapters/outbound/telemetry"
	"github.com/claudioed/facility-layout/internal/application/usecases"
)

// The business counter is only useful if the outcome attribute actually
// reaches the exported data point, so this test collects through a real SDK
// reader rather than asserting against a mock meter.
func TestLocationMetricsCountsRegistrationsByOutcome(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown reported an error: %v", err)
		}
	})

	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	metrics, err := telemetry.NewLocationMetrics()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	metrics.LocationSlotRegistered(ctx, usecases.OutcomeAccepted)
	metrics.LocationSlotRegistered(ctx, usecases.OutcomeAccepted)
	metrics.LocationSlotRegistered(ctx, usecases.OutcomeRejectedByPlacementRule)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &collected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counts := countsByOutcome(t, collected, "facility.location_slot.registrations")
	if counts[usecases.OutcomeAccepted] != 2 {
		t.Fatalf("expected 2 accepted, got %d (%v)", counts[usecases.OutcomeAccepted], counts)
	}
	if counts[usecases.OutcomeRejectedByPlacementRule] != 1 {
		t.Fatalf("expected 1 placement-rule rejection, got %d (%v)", counts[usecases.OutcomeRejectedByPlacementRule], counts)
	}
}

func countsByOutcome(t *testing.T, collected metricdata.ResourceMetrics, name string) map[string]int64 {
	t.Helper()

	counts := map[string]int64{}
	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("%s is not an int64 sum, got %T", name, m.Data)
			}
			for _, point := range sum.DataPoints {
				outcome, ok := point.Attributes.Value(attribute.Key("outcome"))
				if !ok {
					t.Fatalf("data point has no outcome attribute: %v", point.Attributes)
				}
				counts[outcome.AsString()] += point.Value
			}
		}
	}
	if len(counts) == 0 {
		t.Fatalf("metric %q was never exported", name)
	}
	return counts
}
