package mcp

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer builds the MCP server for this bounded context with every read
// tool, the layout resource, and the explore_layout prompt registered.
//
// facility-layout is a read-only Open Host Service, so no write tool is
// registered — only reads over the warehouse map.
func NewServer(deps Deps) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "facility-layout-mcp", Version: "1.0.0"},
		&mcp.ServerOptions{
			Instructions: "Read-only access to the facility-layout warehouse map: the site -> zone -> aisle -> slot structure other contexts consume but never write. List sites, read a site's drawable layout, or a zone's grid. Start with the explore_layout prompt.",
		},
	)

	deps.registerTools(server)
	deps.registerResources(server)
	deps.registerPrompts(server)

	return server
}

// Handler returns the Streamable HTTP handler for the MCP server. There is
// no authentication layer in front of it: this deployable is reached only
// from inside the cluster, and the fleet's static-bearer auth rollout was
// removed (see the ADR recorded alongside this change).
func Handler(server *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
}
