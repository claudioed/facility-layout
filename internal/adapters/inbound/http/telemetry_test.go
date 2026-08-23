package http_test

import (
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// recordSpans installs a recording TracerProvider as the global one for
// the duration of the test and returns the recorder. The router resolves
// the global provider per request, so this captures the spans the
// deployed service would export over OTLP.
func recordSpans(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	return recorder
}

func findAttr(t *testing.T, attrs []attribute.KeyValue, key string) attribute.Value {
	t.Helper()
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return kv.Value
		}
	}
	t.Fatalf("span has no %q attribute, got %v", key, attrs)
	return attribute.Value{}
}

// Span names must be the chi ROUTE PATTERN, never the raw path: one span
// name per location code would make the trace backend useless.
func TestHTTPRequestsProduceSpansNamedForTheRoutePattern(t *testing.T) {
	recorder := recordSpans(t)
	ts := newTestServer(t)

	ts.seedStorageAisle()
	ts.seedSlot("WH1-STOR-AMB-A07-03-02-B", "PalletRack")
	ts.do(http.MethodGet, "/locations/WH1-STOR-AMB-A07-03-02-B", nil).assertStatus(t, http.StatusOK)

	var span sdktrace.ReadOnlySpan
	for _, recorded := range recorder.Ended() {
		if recorded.Name() == "/locations/{locationCode}" {
			span = recorded
		}
	}
	if span == nil {
		t.Fatalf("no span named for the route pattern; got %v", spanNames(recorder))
	}

	attrs := span.Attributes()
	if got := findAttr(t, attrs, "http.method").AsString(); got != "GET" {
		t.Fatalf("expected http.method GET, got %q", got)
	}
	if got := findAttr(t, attrs, "http.status_code").AsInt64(); got != 200 {
		t.Fatalf("expected http.status_code 200, got %d", got)
	}
	if got := findAttr(t, attrs, "http.route").AsString(); got != "/locations/{locationCode}" {
		t.Fatalf("expected the route pattern in http.route, got %q", got)
	}
}

// A failed request has to be findable as a failure in the trace backend,
// not just as a 404 buried in an attribute.
func TestFailedRequestsProduceSpansCarryingTheStatus(t *testing.T) {
	recorder := recordSpans(t)
	ts := newTestServer(t)

	ts.do(http.MethodGet, "/locations/WH1-STOR-AMB-A07-99-99-Z", nil).assertStatus(t, http.StatusNotFound)

	ended := recorder.Ended()
	if len(ended) == 0 {
		t.Fatal("no spans recorded")
	}
	span := ended[len(ended)-1]

	if got := findAttr(t, span.Attributes(), "http.status_code").AsInt64(); got != 404 {
		t.Fatalf("expected http.status_code 404, got %d", got)
	}
	if span.Status().Code == codes.Ok {
		t.Fatalf("expected a non-Ok span status for a 404, got %v", span.Status())
	}
}

func spanNames(recorder *tracetest.SpanRecorder) []string {
	names := make([]string, 0)
	for _, span := range recorder.Ended() {
		names = append(names, span.Name())
	}
	return names
}

// http.server.request.duration is the OTel HTTP semantic-convention
// histogram. It comes from otelchi's metric middleware rather than being
// hand-rolled, so this test is really pinning that it is actually wired.
func TestHTTPRequestsRecordTheServerDurationHistogram(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(t.Context()); err != nil {
			t.Errorf("shutdown reported an error: %v", err)
		}
	})

	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	// The router resolves its meter when it is built, so it must be built
	// after the provider is installed.
	ts := newTestServer(t)
	ts.do(http.MethodGet, "/healthz", nil).assertStatus(t, http.StatusOK)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != "http.server.request.duration" {
				continue
			}
			histogram, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("expected a float64 histogram, got %T", m.Data)
			}
			if len(histogram.DataPoints) == 0 || histogram.DataPoints[0].Count == 0 {
				t.Fatalf("the histogram recorded nothing: %v", histogram)
			}
			if m.Unit != "s" {
				t.Fatalf("expected seconds per the HTTP semantic conventions, got %q", m.Unit)
			}
			return
		}
	}
	t.Fatal("http.server.request.duration was never recorded")
}
