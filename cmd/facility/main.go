// Command facility is the composition root: it wires env config into
// adapters, adapters into use cases, and use cases into the HTTP router.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	inboundhttp "github.com/claudioed/facility-layout/internal/adapters/inbound/http"
	"github.com/claudioed/facility-layout/internal/adapters/outbound/events"
	"github.com/claudioed/facility-layout/internal/adapters/outbound/memory"
	"github.com/claudioed/facility-layout/internal/adapters/outbound/postgres"
	"github.com/claudioed/facility-layout/internal/application/ports"
	"github.com/claudioed/facility-layout/internal/application/usecases"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	httpAddr := getenv("HTTP_ADDR", ":8080")
	databaseURL := os.Getenv("DATABASE_URL")
	migrationsPath := getenv("MIGRATIONS_PATH", "migrations")

	logger := log.New(os.Stdout, "facility-layout ", log.LstdFlags)

	adapters, closeAdapters, err := buildAdapters(databaseURL, migrationsPath, logger)
	if err != nil {
		return err
	}
	defer closeAdapters()

	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           inboundhttp.NewRouter(newServer(adapters, memory.SystemClock{})),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("listening on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

// adapterSet is the concrete outbound side of the hexagon, chosen at
// startup: Postgres when DATABASE_URL is set, in-memory otherwise.
type adapterSet struct {
	sites         ports.SiteRepo
	zones         ports.ZoneRepo
	aisles        ports.AisleRepo
	slots         ports.SlotRepo
	locationTypes ports.LocationTypeRepo
	rules         ports.PlacementRuleRepo
	publisher     ports.EventPublisher
}

// newServer wires every use case over the chosen adapters. It is the one
// place in the codebase that knows about all three layers at once.
func newServer(a adapterSet, clock ports.Clock) *inboundhttp.Server {
	return &inboundhttp.Server{
		RegisterSite: &usecases.RegisterSite{Sites: a.sites, Events: a.publisher, Clock: clock},
		GetSite:      &usecases.GetSite{Sites: a.sites},
		ListSites:    &usecases.ListSites{Sites: a.sites},

		RegisterZone: &usecases.RegisterZone{Sites: a.sites, Zones: a.zones, Events: a.publisher, Clock: clock},
		GetZone:      &usecases.GetZone{Zones: a.zones},
		ListZones:    &usecases.ListZones{Sites: a.sites, Zones: a.zones},

		RegisterAisle: &usecases.RegisterAisle{Zones: a.zones, Aisles: a.aisles, Events: a.publisher, Clock: clock},
		GetAisle:      &usecases.GetAisle{Aisles: a.aisles},
		ListAisles:    &usecases.ListAisles{Zones: a.zones, Aisles: a.aisles},

		RegisterLocationType: &usecases.RegisterLocationType{LocationTypes: a.locationTypes, Events: a.publisher, Clock: clock},
		GetLocationType:      &usecases.GetLocationType{LocationTypes: a.locationTypes},
		ListLocationTypes:    &usecases.ListLocationTypes{LocationTypes: a.locationTypes},

		DefinePlacementRule: &usecases.DefinePlacementRule{LocationTypes: a.locationTypes, Rules: a.rules, Events: a.publisher, Clock: clock},
		GetPlacementRule:    &usecases.GetPlacementRule{Rules: a.rules},
		ListPlacementRules:  &usecases.ListPlacementRules{Rules: a.rules},

		RegisterLocationSlot: &usecases.RegisterLocationSlot{
			Sites: a.sites, Zones: a.zones, Aisles: a.aisles, Slots: a.slots,
			LocationTypes: a.locationTypes, Rules: a.rules, Events: a.publisher, Clock: clock,
		},
		GetLocationSlot:          &usecases.GetLocationSlot{Slots: a.slots},
		DecommissionLocationSlot: &usecases.DecommissionLocationSlot{Slots: a.slots, Events: a.publisher, Clock: clock},
		ImportFacilityLayout: &usecases.ImportFacilityLayout{
			Sites: a.sites, Zones: a.zones, Aisles: a.aisles, Slots: a.slots,
			LocationTypes: a.locationTypes, Rules: a.rules, Events: a.publisher, Clock: clock,
		},

		GetSiteLayout: &usecases.GetSiteLayout{Sites: a.sites, Zones: a.zones, Aisles: a.aisles, Slots: a.slots},
		GetZoneGrid:   &usecases.GetZoneGrid{Zones: a.zones, Aisles: a.aisles, Slots: a.slots},
	}
}

// buildAdapters wires the Postgres adapters when DATABASE_URL is set, or
// falls back to the in-memory adapters for local development without a
// database.
func buildAdapters(databaseURL, migrationsPath string, logger *log.Logger) (adapterSet, func(), error) {
	noop := func() {}

	if databaseURL == "" {
		logger.Println("DATABASE_URL not set; using in-memory adapters")
		return adapterSet{
			sites:         memory.NewSiteRepo(),
			zones:         memory.NewZoneRepo(),
			aisles:        memory.NewAisleRepo(),
			slots:         memory.NewSlotRepo(),
			locationTypes: memory.NewLocationTypeRepo(),
			rules:         memory.NewPlacementRuleRepo(),
			publisher:     events.NewLogPublisher(logger),
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
		sites:         postgres.NewSiteRepo(pool),
		zones:         postgres.NewZoneRepo(pool),
		aisles:        postgres.NewAisleRepo(pool),
		slots:         postgres.NewSlotRepo(pool),
		locationTypes: postgres.NewLocationTypeRepo(pool),
		rules:         postgres.NewPlacementRuleRepo(pool),
		publisher:     postgres.NewEventPublisher(pool),
	}, pool.Close, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
