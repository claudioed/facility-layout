// Package kafka contains the inbound Kafka adapters. Alongside the OLTP
// consumer, analytics_consumer.go consumes the analytics topic and projects the
// facility-layout "Layout Catalog Growth & Change" read model.
//
// Consistent with the ADR-0009 integration publisher and the ADR-0010 analytics
// publisher, the analytics pipeline is trace-free: facility-layout has no
// observability/OTel package, so this consumer opens no spans and reads no trace
// headers.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/claudioed/facility-layout/internal/analytics/report"
)

// AnalyticsConsumerGroup is the Kafka consumer group the analytics projector
// reads under. It is distinct from any OLTP consumer group so the two pipelines
// track their offsets independently.
const AnalyticsConsumerGroup = "facility-analytics"

// ProcessedEvents is the consumer's idempotency gate: MarkProcessed records an
// event id if it has not been seen and reports whether this call was the first
// to record it. It is declared here (rather than in application/ports) because
// it is an analytics-only concern the OLTP layers never touch; the analyticsstore
// ConsumedEventsRepo implements it.
type ProcessedEvents interface {
	MarkProcessed(ctx context.Context, eventId string) (bool, error)
}

// analyticsEnvelope is the inbound decode shape of the Envelope v1 wrapper on
// the analytics topic. The data payload is left as a RawMessage and decoded per
// event_type. It is declared here (rather than imported from the outbound
// publisher) so this inbound adapter does not depend on an outbound adapter.
type analyticsEnvelope struct {
	EventId       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Source        string          `json:"source"`
	SchemaVersion int             `json:"schema_version"`
	Data          json.RawMessage `json:"data"`
}

// analyticsData is the union of fields the projecting event payloads carry (the
// domain events' own camelCase JSON, forwarded verbatim as the envelope data).
// Each event_type populates the subset it needs.
type analyticsData struct {
	SiteCode      string `json:"siteCode"`
	ZoneID        string `json:"zoneId"`
	AisleID       string `json:"aisleId"`
	LocationCode  string `json:"locationCode"`
	LocationType  string `json:"locationType"`
	RuleID        string `json:"ruleId"`
	RowsSubmitted int    `json:"rowsSubmitted"`
	SlotsImported int    `json:"slotsImported"`
	RowsRejected  int    `json:"rowsRejected"`
}

// AnalyticsConsumer reads analytics events off the analytics topic and applies
// each to the catalog-growth ProjectionStore, exactly once per event_id despite
// Kafka's at-least-once delivery.
type AnalyticsConsumer struct {
	Reader     *kafkago.Reader
	Projection report.ProjectionStore
	Processed  ProcessedEvents
	Logger     *slog.Logger
}

// NewAnalyticsConsumer constructs an AnalyticsConsumer reading topic from
// brokers under AnalyticsConsumerGroup.
func NewAnalyticsConsumer(brokers []string, topic string, projection report.ProjectionStore, processed ProcessedEvents, logger *slog.Logger) *AnalyticsConsumer {
	if logger == nil {
		logger = slog.Default()
	}
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: AnalyticsConsumerGroup,
		// Start a brand-new consumer group at the EARLIEST offset. The analytics
		// projection must see the full history of the topic (it is a replayable
		// read model, not a live integration reaction), so a fresh projector — or
		// a backfill into a new group — reads from the beginning rather than
		// kafka-go's default of the latest offset, which would silently drop
		// every event produced before the group first committed an offset. Once
		// the group has committed offsets, those take precedence and this only
		// affects the first join.
		StartOffset: kafkago.FirstOffset,
	})
	return &AnalyticsConsumer{Reader: reader, Projection: projection, Processed: processed, Logger: logger}
}

// Run reads and handles messages until ctx is cancelled or the reader returns a
// fatal error. A handling error is logged and the loop continues so one bad
// message cannot wedge the projector.
func (c *AnalyticsConsumer) Run(ctx context.Context) error {
	for {
		msg, err := c.Reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if err := c.HandleMessage(ctx, msg.Value); err != nil {
			c.Logger.ErrorContext(ctx, "analytics message handling failed", "error", err)
		}
	}
}

// Close releases the underlying Kafka reader.
func (c *AnalyticsConsumer) Close() error {
	return c.Reader.Close()
}

// HandleMessage decodes raw as an analyticsEnvelope and applies the matching
// projection method for its event_type. Event types outside the projection
// contract are ignored (and not marked processed). For a projecting event it
// dedupes on event_id via ProcessedEvents before applying, so a redelivery is a
// no-op. It is exported separately from Run so tests can feed raw envelopes
// without a live broker.
func (c *AnalyticsConsumer) HandleMessage(ctx context.Context, raw []byte) error {
	var env analyticsEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("analytics: decode envelope: %w", err)
	}

	// The report is derived from the catalog-change event set. Any other
	// event_type is acknowledged without touching the read model or the
	// processed set, so a later contract change could still reprocess it.
	name := eventName(env.EventType)
	switch name {
	case "SiteRegistered", "ZoneRegistered", "AisleRegistered",
		"LocationTypeRegistered", "PlacementRuleDefined",
		"LocationSlotRegistered", "LocationSlotDecommissioned",
		"FacilityLayoutImported":
	default:
		return nil
	}

	isNew, err := c.Processed.MarkProcessed(ctx, env.EventId)
	if err != nil {
		return fmt.Errorf("analytics: mark processed: %w", err)
	}
	if !isNew {
		return nil
	}

	var data analyticsData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return fmt.Errorf("analytics: decode data: %w", err)
	}

	switch name {
	case "SiteRegistered":
		return c.Projection.ApplySiteRegistered(ctx, env.EventId, data.SiteCode, env.OccurredAt)
	case "ZoneRegistered":
		// A zone is a growth of its site, so it is scoped to the site code.
		return c.Projection.ApplyZoneRegistered(ctx, env.EventId, data.SiteCode, env.OccurredAt)
	case "AisleRegistered":
		return c.Projection.ApplyAisleRegistered(ctx, env.EventId, data.ZoneID, env.OccurredAt)
	case "LocationTypeRegistered":
		// A location type is a catalog-wide definition, not scoped to a site or
		// zone: it lands in the empty catalog-wide scope.
		return c.Projection.ApplyLocationTypeRegistered(ctx, env.EventId, "", env.OccurredAt)
	case "PlacementRuleDefined":
		return c.Projection.ApplyPlacementRuleDefined(ctx, env.EventId, "", env.OccurredAt)
	case "LocationSlotRegistered":
		return c.Projection.ApplyLocationSlotRegistered(ctx, env.EventId, zoneOf(data.LocationCode), env.OccurredAt)
	case "LocationSlotDecommissioned":
		return c.Projection.ApplyLocationSlotDecommissioned(ctx, env.EventId, zoneOf(data.LocationCode), env.OccurredAt)
	case "FacilityLayoutImported":
		return c.Projection.ApplyFacilityLayoutImported(ctx, env.EventId, "", data.RowsSubmitted, data.SlotsImported, data.RowsRejected, env.OccurredAt)
	default:
		return nil
	}
}

// eventName extracts the trailing PascalCase event name from a CloudEvents-style
// type such as "com.warehouse.wms.facility-layout.slot.LocationSlotRegistered".
// If the type carries no dot it is returned unchanged, so a bare event name
// (as some tests use) still matches.
func eventName(eventType string) string {
	if i := strings.LastIndex(eventType, "."); i >= 0 {
		return eventType[i+1:]
	}
	return eventType
}

// zoneOf derives the zone id (SITE-AREA-ZONE) from a full location code by
// keeping its first three hyphen-joined segments. This mirrors the domain's
// LocationCode.ZoneID() without importing the domain: the consumer stays free of
// any OLTP dependency. An unexpectedly short code is returned as-is.
func zoneOf(locationCode string) string {
	parts := strings.Split(locationCode, "-")
	if len(parts) < 3 {
		return locationCode
	}
	return strings.Join(parts[:3], "-")
}
