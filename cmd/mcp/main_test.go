package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/claudioed/facility-layout/internal/adapters/inbound/auth"
	inboundmcp "github.com/claudioed/facility-layout/internal/adapters/inbound/mcp"
	"github.com/claudioed/facility-layout/internal/adapters/outbound/memory"
	"github.com/claudioed/facility-layout/internal/application/usecases"
)

// testRouter builds the exact HTTP surface cmd/mcp serves: the real MCP
// server over in-memory repos, the real static-key auth, wrapped by newRouter.
func testRouter(t *testing.T) http.Handler {
	t.Helper()
	sites := memory.NewSiteRepo()
	zones := memory.NewZoneRepo()
	aisles := memory.NewAisleRepo()
	slots := memory.NewSlotRepo()
	server := inboundmcp.NewServer(inboundmcp.Deps{
		GetSiteLayout: &usecases.GetSiteLayout{Sites: sites, Zones: zones, Aisles: aisles, Slots: slots},
		GetZoneGrid:   &usecases.GetZoneGrid{Zones: zones, Aisles: aisles, Slots: slots},
		ListSites:     &usecases.ListSites{Sites: sites},
	})
	authn := auth.NewStaticKeyAuth(map[string]auth.Scope{"read-key": auth.ScopeRead})
	return newRouter(inboundmcp.Handler(server, authn))
}

func TestHealthz_IsUnauthenticated(t *testing.T) {
	rec := httptest.NewRecorder()
	testRouter(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz without a bearer key: got %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("GET /healthz body: got %q", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("GET /healthz Content-Type: got %q", ct)
	}
}

func TestMCPMounts_RequireBearerKey(t *testing.T) {
	for _, path := range []string{"/", "/mcp"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			testRouter(t).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("POST %s without a bearer key: got %d, want 401", path, rec.Code)
			}
			if got := rec.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, "Bearer ") {
				t.Fatalf("POST %s: missing WWW-Authenticate challenge, got %q", path, got)
			}
		})
	}
}

func TestMCPMounts_ReachStreamableHandlerWithBearerKey(t *testing.T) {
	// With a valid key the request must get PAST the auth middleware and into
	// the Streamable HTTP handler. An initialize request is the one call a
	// fresh session may make, so a JSON-RPC result (not 401/404) proves the
	// handler is mounted at that path.
	const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	for _, path := range []string{"/", "/mcp"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(initialize))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			req.Header.Set("Authorization", "Bearer read-key")
			rec := httptest.NewRecorder()
			testRouter(t).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("POST %s initialize with a valid key: got %d, want 200 (body %q)", path, rec.Code, rec.Body.String())
			}
			body, _ := io.ReadAll(rec.Body)
			if !strings.Contains(string(body), `"serverInfo"`) {
				t.Fatalf("POST %s initialize: expected a JSON-RPC initialize result, got %q", path, body)
			}
		})
	}
}

func TestUnknownPathUnderMCPPrefix_StillAuthenticated(t *testing.T) {
	// chi's Mount also covers /mcp/*; the sub-path must not bypass auth.
	rec := httptest.NewRecorder()
	testRouter(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcp/anything", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /mcp/anything without a key: got %d, want 401", rec.Code)
	}
}
