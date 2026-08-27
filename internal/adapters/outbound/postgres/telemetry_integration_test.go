//go:build integration

package postgres_test

import (
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/claudioed/facility-layout/internal/adapters/outbound/postgres"
	"github.com/claudioed/facility-layout/internal/domain/site"
)

// Every database call must land under the span of whatever use case made
// it — an orphan DB span tells you a query was slow but not which request
// paid for it. This needs a live Postgres, hence the integration tag.
func TestDatabaseCallsBecomeChildSpans(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	// The pool installs its otelpgx tracer at construction, so it must be
	// opened after the recording provider is the global one.
	ctx, pool := newPool(t)

	ctx, parent := provider.Tracer("test").Start(ctx, "register site")
	s, err := site.NewSite("WHX", "Span Probe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := postgres.NewSiteRepo(pool).Save(ctx, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parent.End()

	var queried sdktrace.ReadOnlySpan
	for _, span := range recorder.Ended() {
		if strings.HasPrefix(span.Name(), "query ") {
			queried = span
		}
	}
	if queried == nil {
		t.Fatal("no query span was recorded for the INSERT")
	}
	if !queried.Parent().IsValid() {
		t.Fatal("the query span is an orphan; it must hang off the calling use case's span")
	}
	if queried.SpanContext().TraceID() != parent.SpanContext().TraceID() {
		t.Fatalf("the query span is in a different trace from its use case: %v vs %v",
			queried.SpanContext().TraceID(), parent.SpanContext().TraceID())
	}

	// The statement is recorded normalized: placeholders, never literals.
	// This is what keeps span attributes free of customer data and of
	// unbounded cardinality.
	var statement string
	for _, attr := range queried.Attributes() {
		if attr.Key == "db.query.text" {
			statement = attr.Value.AsString()
		}
	}
	if statement == "" {
		t.Fatalf("the query span carries no db.query.text: %v", queried.Attributes())
	}
	if !strings.Contains(statement, "$1") {
		t.Fatalf("expected a parameterized statement, got %q", statement)
	}
	if strings.Contains(statement, "Span Probe") {
		t.Fatalf("the span leaked a literal value: %q", statement)
	}
}
