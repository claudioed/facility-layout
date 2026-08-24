// Command mcp is the composition root for the Facility Layout MCP server: it
// wires env config to outbound adapters, adapters to the read use cases, and
// those to the inbound MCP adapter, then serves MCP over Streamable HTTP. It
// is a second, independent deployable alongside cmd/facility (the HTTP
// service), per ADR-0007.
//
// facility-layout is a read-only Open Host Service, so this server wires only
// the read use cases (GetSiteLayout, GetZoneGrid, ListSites) and exposes only
// read tools.
//
// Auth is a static bearer key (no IdP): set MCP_READ_KEY (and, for the future
// write seam, MCP_READWRITE_KEY) from a Kubernetes Secret. A request must
// present a valid key; the scope it grants gates the tools.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	inboundmcp "github.com/claudioed/facility-layout/internal/adapters/inbound/mcp"
	"github.com/claudioed/facility-layout/internal/adapters/outbound/memory"
	"github.com/claudioed/facility-layout/internal/adapters/outbound/postgres"
	"github.com/claudioed/facility-layout/internal/adapters/outbound/telemetry"
	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/application/usecases"
)

func main() {
	if err := run(); err != nil {
		slog.Error("mcp server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	serviceName := getenv("OTEL_SERVICE_NAME", "facility-layout-mcp")

	logger := telemetry.NewLogger(os.Stdout, getenv("LOG_LEVEL", "info"), serviceName)
	slog.SetDefault(logger)

	// Same non-blocking telemetry setup as the HTTP service: an unreachable
	// Collector degrades to dropped telemetry, never a server that won't start.
	shutdownTelemetry, err := telemetry.Setup(
		context.Background(),
		serviceName,
		getenv("SERVICE_VERSION", "dev"),
		getenv("OTEL_EXPORTER_OTLP_ENDPOINT", telemetry.DefaultOTLPEndpoint),
	)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			logger.Warn("telemetry shutdown reported an error", "error", err)
		}
	}()

	httpAddr := getenv("MCP_ADDR", ":8090")
	databaseURL := os.Getenv("DATABASE_URL")
	migrationsPath := getenv("MIGRATIONS_PATH", "migrations")

	adapters, closeAdapters, err := buildAdapters(databaseURL, migrationsPath, logger)
	if err != nil {
		return err
	}
	defer closeAdapters()

	// The MCP adapter reuses the SAME read use cases the HTTP adapter uses:
	// GetSiteLayout, GetZoneGrid and ListSites, each over the same repos. No
	// write use case is wired — this context's map is consumed, not mutated.
	deps := inboundmcp.Deps{
		GetSiteLayout: &usecases.GetSiteLayout{Sites: adapters.sites, Zones: adapters.zones, Aisles: adapters.aisles, Slots: adapters.slots},
		GetZoneGrid:   &usecases.GetZoneGrid{Zones: adapters.zones, Aisles: adapters.aisles, Slots: adapters.slots},
		ListSites:     &usecases.ListSites{Sites: adapters.sites},
	}
	server := inboundmcp.NewServer(deps)

	auth := inboundmcp.NewStaticKeyAuth(authKeys(logger))
	handler := inboundmcp.Handler(server, auth)

	srv := &http.Server{Addr: httpAddr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		logger.Info("mcp server listening (Streamable HTTP)", "addr", httpAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("mcp server failed", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// adapterSet is the read-side outbound repos the MCP server needs, chosen at
// startup: Postgres when DATABASE_URL is set, in-memory otherwise. It mirrors
// cmd/facility's selection so both binaries read the same map.
type adapterSet struct {
	sites  ports.SiteRepo
	zones  ports.ZoneRepo
	aisles ports.AisleRepo
	slots  ports.SlotRepo
}

// buildAdapters wires the Postgres repos when DATABASE_URL is set, or falls
// back to the in-memory repos for local development without a database —
// exactly as cmd/facility/main.go does.
func buildAdapters(databaseURL, migrationsPath string, logger *slog.Logger) (adapterSet, func(), error) {
	noop := func() {}

	if databaseURL == "" {
		logger.Info("database url not configured; using in-memory adapters")
		return adapterSet{
			sites:  memory.NewSiteRepo(),
			zones:  memory.NewZoneRepo(),
			aisles: memory.NewAisleRepo(),
			slots:  memory.NewSlotRepo(),
		}, noop, nil
	}

	if err := postgres.RunMigrations(databaseURL, migrationsPath); err != nil {
		return adapterSet{}, noop, err
	}

	pool, err := postgres.NewPool(context.Background(), databaseURL)
	if err != nil {
		return adapterSet{}, noop, err
	}

	return adapterSet{
		sites:  postgres.NewSiteRepo(pool),
		zones:  postgres.NewZoneRepo(pool),
		aisles: postgres.NewAisleRepo(pool),
		slots:  postgres.NewSlotRepo(pool),
	}, pool.Close, nil
}

// authKeys reads the bearer keys from the environment. MCP_READ_KEY grants
// read scope; MCP_READWRITE_KEY grants read-write (kept for the future write
// seam even though no write tool is registered yet). If neither is set the
// server still starts but rejects every request (fail closed) — a missing key
// must never mean "open to everyone". The keys themselves are never logged.
func authKeys(logger *slog.Logger) map[string]inboundmcp.Scope {
	keys := make(map[string]inboundmcp.Scope)
	if k := os.Getenv("MCP_READ_KEY"); k != "" {
		keys[k] = inboundmcp.ScopeRead
	}
	if k := os.Getenv("MCP_READWRITE_KEY"); k != "" {
		keys[k] = inboundmcp.ScopeReadWrite
	}
	if len(keys) == 0 {
		logger.Warn("no MCP_READ_KEY or MCP_READWRITE_KEY set; server will reject all requests")
	}
	return keys
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
