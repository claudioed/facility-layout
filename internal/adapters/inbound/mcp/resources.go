package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// layoutURIScheme is the scheme+authority prefix of the site-layout resource
// URI. A concrete resource URI is layoutURIScheme + "<siteCode>".
const layoutURIScheme = "layout://facility/"

// registerResources adds the scoped read-model resource. Per the charter,
// resources are bounded-context contracts tied to a decision, not bulk dumps:
// the layout resource answers "what is the drawable structure of this one
// site?", backed by the same GetSiteLayout read model the tool uses.
//
// The resource is registered as a template (layout://facility/{siteCode}) so a
// client can read any site's layout by URI without a per-site registration.
func (d Deps) registerResources(server *mcp.Server, scopeOf func(context.Context) Scope) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: layoutURIScheme + "{siteCode}",
		Name:        "site layout",
		Description: "The full drawable structure of one site (zones -> aisles -> slot codes), addressed by site code, e.g. layout://facility/WH1.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		if !scopeAllows(scopeOf(ctx), ScopeRead) {
			return nil, fmt.Errorf("resource %q requires read scope", uri)
		}
		siteCode, ok := strings.CutPrefix(uri, layoutURIScheme)
		if !ok || siteCode == "" {
			return nil, fmt.Errorf("resource %q is not a valid site-layout URI", uri)
		}
		layout, err := d.GetSiteLayout.Execute(ctx, siteCode)
		if err != nil {
			return nil, err
		}
		body, err := json.Marshal(toSiteLayoutDTO(layout))
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(body),
			}},
		}, nil
	})
}
