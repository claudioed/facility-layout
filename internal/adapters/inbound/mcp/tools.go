package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/claudioed/facility-layout/internal/application/usecases"
)

// tracerName is the OTel instrumentation scope for MCP tool spans.
const tracerName = "github.com/claudioed/facility-layout/internal/adapters/inbound/mcp"

// Deps is everything the MCP tools need, injected by the composition root.
// It carries the SAME read use cases the HTTP adapter uses; the adapter never
// constructs an outbound adapter itself.
//
// facility-layout is read-only to the rest of the system, so every dependency
// here is a read-model use case. There is no write use case and no write tool.
type Deps struct {
	// GetSiteLayout is the existing read-model use case behind get_site_layout
	// and the layout resource, reused unchanged.
	GetSiteLayout *usecases.GetSiteLayout
	// GetZoneGrid is the existing read-model use case behind get_zone_grid.
	GetZoneGrid *usecases.GetZoneGrid
	// ListSites is the existing read use case behind list_sites.
	ListSites *usecases.ListSites
}

// --- get_site_layout ----------------------------------------------------------

type siteLayoutInput struct {
	SiteCode string `json:"siteCode" jsonschema:"the code of the site whose layout to return, e.g. WH1"`
}

func (d Deps) getSiteLayout(ctx context.Context, in siteLayoutInput) (siteLayoutDTO, error) {
	if in.SiteCode == "" {
		return siteLayoutDTO{}, fmt.Errorf("siteCode is required")
	}
	layout, err := d.GetSiteLayout.Execute(ctx, in.SiteCode)
	if err != nil {
		return siteLayoutDTO{}, err
	}
	return toSiteLayoutDTO(layout), nil
}

// --- get_zone_grid ------------------------------------------------------------

type zoneGridInput struct {
	ZoneID string `json:"zoneId" jsonschema:"the id of the zone whose drawable grid to return, e.g. WH1-STOR-AMB"`
}

func (d Deps) getZoneGrid(ctx context.Context, in zoneGridInput) (zoneGridDTO, error) {
	if in.ZoneID == "" {
		return zoneGridDTO{}, fmt.Errorf("zoneId is required")
	}
	grid, err := d.GetZoneGrid.Execute(ctx, in.ZoneID)
	if err != nil {
		return zoneGridDTO{}, err
	}
	return toZoneGridDTO(grid), nil
}

// --- list_sites ---------------------------------------------------------------

type listSitesInput struct{}

type listSitesOutput struct {
	Sites []siteRef `json:"sites"`
}

func (d Deps) listSites(ctx context.Context, _ listSitesInput) (listSitesOutput, error) {
	sites, err := d.ListSites.Execute(ctx)
	if err != nil {
		return listSitesOutput{}, err
	}
	out := listSitesOutput{Sites: make([]siteRef, 0, len(sites))}
	for _, s := range sites {
		out.Sites = append(out.Sites, toSiteRef(s))
	}
	return out, nil
}

// --- registration -------------------------------------------------------------

// registerTools adds every tool to the server, each wrapped so its handler
// runs inside an OTel span named "mcp.tool <name>" and is gated by the
// session's scope.
//
// facility-layout is a read-only Open Host Service: every registered tool is a
// read tool and requires ScopeRead. No write tool is registered (the map is
// consumed, not mutated). The scope-parameterised addTool wrapper is kept
// identical to the pilot so a future write tool can be added with
// ScopeReadWrite and no other change.
func (d Deps) registerTools(server *mcp.Server, scopeOf func(context.Context) Scope) {
	readOnly := true

	addTool(server, scopeOf, ScopeRead, &mcp.Tool{
		Name:        "list_sites",
		Description: "List every registered site on the warehouse map as {code, name}. Start here to discover which sites exist before drilling into a layout.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly},
	}, d.listSites)

	addTool(server, scopeOf, ScopeRead, &mcp.Tool{
		Name:        "get_site_layout",
		Description: "Return one site's full drawable structure as a compact nested map: zones -> aisles -> slot codes, with each zone's temperature class and hazmat flag and each aisle's walk-order sequence hint. Use it to answer placement and travel questions for a site.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly},
	}, d.getSiteLayout)

	addTool(server, scopeOf, ScopeRead, &mcp.Tool{
		Name:        "get_zone_grid",
		Description: "Return one zone's slots as a 2D grid: rows are levels, columns are (aisle, bay) pairs in walk order, each cell holds the location codes at that coordinate. Use it to reason about a single zone's rack layout in detail.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly},
	}, d.getZoneGrid)
}

// addTool registers one scope-gated tool. It centralises the cross-cutting
// concerns every tool shares: a span per call, scope enforcement against the
// tool's required minimum scope, and mapping a handler error onto the span
// before returning it. It is parameterised on the required scope so a future
// write tool (ScopeReadWrite) reuses it unchanged.
func addTool[In, Out any](
	server *mcp.Server,
	scopeOf func(context.Context) Scope,
	required Scope,
	tool *mcp.Tool,
	handle func(context.Context, In) (Out, error),
) {
	mcp.AddTool(server, tool, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var zero Out
		ctx, span := otel.Tracer(tracerName).Start(ctx, "mcp.tool "+tool.Name,
			trace.WithAttributes(
				attribute.String("mcp.tool.name", tool.Name),
				attribute.String("mcp.tool.required_scope", string(required)),
			),
		)
		defer span.End()

		if !scopeAllows(scopeOf(ctx), required) {
			err := fmt.Errorf("tool %q requires %s scope", tool.Name, required)
			span.SetStatus(codes.Error, "unauthorized")
			return nil, zero, err
		}

		out, err := handle(ctx, in)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, zero, err
		}
		return nil, out, nil
	})
}
