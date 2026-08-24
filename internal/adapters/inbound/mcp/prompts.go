package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// exploreLayoutSOP is the operational standard-operating-procedure the
// explore_layout prompt hands to the model. Per the charter, prompts encode
// how to navigate the map, how to interpret results, and what "done" means —
// they standardise agent behaviour across clients rather than leaving
// procedure implicit.
const exploreLayoutSOP = `You are exploring the facility-layout warehouse map to answer a placement or travel question. Use only the MCP tools; never assume structure. This context is READ-ONLY — the map is a source of truth you consume, not change.

Procedure (navigate coarsest to finest: site -> zone -> aisle -> slot):
1. Call list_sites to see which sites exist. Pick the site the question concerns by its code.
2. Call get_site_layout for that site code. This returns the whole site as zones -> aisles -> slot codes, with each zone's temperatureClass and hazmat flag and each aisle's sequenceHint (its walk-order position).
3. To reason about ONE zone's rack in detail, call get_zone_grid with that zone's id (e.g. WH1-STOR-AMB). Rows are levels, columns are (aisle, bay) pairs in walk order, cells hold the location codes at each coordinate.

Interpretation:
- A slot's location code is Site-Area-Zone-Aisle-Bay-Level-Position; read it left-to-right to place it physically.
- A zone's temperatureClass and hazmat flag decide what storage is legal there — an ambient item does not belong in a FRZ (frozen) zone.
- Aisle sequenceHint is walk order: lower comes earlier on a pick route, so travel distance between two aisles tracks the gap in their hints.

Answering a travel question: get the two aisles' sequenceHints from get_site_layout (or get_zone_grid columns) and compare them; the walk order, not the aisle code, is what orders the route.

Answering a placement question: find a zone whose temperatureClass/hazmat suit the item, then a slot code in an aisle of that zone from the grid.

Done means: you have named the specific site, zone(s), aisle(s) and slot code(s) that answer the question, each justified from tool output. Do not attempt to change anything; this service exposes no write tool.`

// registerPrompts adds the workflow prompts (operational SOPs).
func (d Deps) registerPrompts(server *mcp.Server, _ func(context.Context) Scope) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "explore_layout",
		Description: "Standard operating procedure for navigating the warehouse map (site -> zone -> aisle -> slot) to answer a placement or travel question using the read tools.",
	}, func(ctx context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "How to explore the layout: list sites, read a site's zones/aisles/slots, drill into a zone grid, and reason about placement and travel — read-only.",
			Messages: []*mcp.PromptMessage{{
				Role:    "user",
				Content: &mcp.TextContent{Text: exploreLayoutSOP},
			}},
		}, nil
	})
}
