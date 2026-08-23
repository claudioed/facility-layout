package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/claudioed/facility-layout/internal/application/usecases"
	"github.com/claudioed/facility-layout/internal/domain/placement"
	"github.com/claudioed/facility-layout/internal/domain/shared"
)

// Server holds every use case the HTTP adapter depends on.
type Server struct {
	RegisterSite             *usecases.RegisterSite
	GetSite                  *usecases.GetSite
	ListSites                *usecases.ListSites
	RegisterZone             *usecases.RegisterZone
	GetZone                  *usecases.GetZone
	ListZones                *usecases.ListZones
	RegisterAisle            *usecases.RegisterAisle
	GetAisle                 *usecases.GetAisle
	ListAisles               *usecases.ListAisles
	RegisterLocationType     *usecases.RegisterLocationType
	GetLocationType          *usecases.GetLocationType
	ListLocationTypes        *usecases.ListLocationTypes
	DefinePlacementRule      *usecases.DefinePlacementRule
	GetPlacementRule         *usecases.GetPlacementRule
	ListPlacementRules       *usecases.ListPlacementRules
	RegisterLocationSlot     *usecases.RegisterLocationSlot
	GetLocationSlot          *usecases.GetLocationSlot
	DecommissionLocationSlot *usecases.DecommissionLocationSlot
	ImportFacilityLayout     *usecases.ImportFacilityLayout
	GetSiteLayout            *usecases.GetSiteLayout
	GetZoneGrid              *usecases.GetZoneGrid
}

// NewRouter builds the chi router for every endpoint in CLAUDE.md's REST
// API section.
//
// Three single-resource GETs (a zone, an aisle, a location type, a
// placement rule) are additive to that list: without them the Location
// header this service is required to set on every 201 would point at a URL
// with no representation, which is not REST maturity level 2.
func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealthz)

	r.Route("/sites", func(r chi.Router) {
		r.Post("/", s.handleRegisterSite)
		r.Get("/", s.handleListSites)
		r.Get("/{siteCode}", s.handleGetSite)
		r.Get("/{siteCode}/layout", s.handleGetSiteLayout)
		r.Post("/{siteCode}/zones", s.handleRegisterZone)
		r.Get("/{siteCode}/zones", s.handleListZones)
	})

	r.Route("/zones", func(r chi.Router) {
		r.Get("/{zoneId}", s.handleGetZone)
		r.Get("/{zoneId}/grid", s.handleGetZoneGrid)
		r.Post("/{zoneId}/aisles", s.handleRegisterAisle)
		r.Get("/{zoneId}/aisles", s.handleListAisles)
		r.Get("/{zoneId}/aisles/{aisleCode}", s.handleGetAisle)
	})

	r.Route("/location-types", func(r chi.Router) {
		r.Post("/", s.handleRegisterLocationType)
		r.Get("/", s.handleListLocationTypes)
		r.Get("/{name}", s.handleGetLocationType)
	})

	r.Route("/placement-rules", func(r chi.Router) {
		r.Post("/", s.handleDefinePlacementRule)
		r.Get("/", s.handleListPlacementRules)
		r.Get("/{ruleId}", s.handleGetPlacementRule)
	})

	r.Route("/locations", func(r chi.Router) {
		r.Post("/", s.handleRegisterLocationSlot)
		r.Post("/import", s.handleImportFacilityLayout)
		r.Get("/{locationCode}", s.handleGetLocationSlot)
		r.Post("/{locationCode}/decommission", s.handleDecommissionLocationSlot)
	})

	return r
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------- sites ----

func (s *Server) handleRegisterSite(w http.ResponseWriter, r *http.Request) {
	var req registerSiteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	registered, err := s.RegisterSite.Execute(r.Context(), req.SiteCode, req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/sites/"+registered.Code())
	writeJSON(w, http.StatusCreated, toSiteResponse(registered))
}

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	sites, err := s.ListSites.Execute(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]siteResponse, 0, len(sites))
	for _, site := range sites {
		out = append(out, toSiteResponse(site))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	found, err := s.GetSite.Execute(r.Context(), chi.URLParam(r, "siteCode"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSiteResponse(found))
}

// ---------------------------------------------------------------- zones ----

func (s *Server) handleRegisterZone(w http.ResponseWriter, r *http.Request) {
	var req registerZoneRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	temperatureClass, err := shared.ParseTemperatureClass(req.TemperatureClass)
	if err != nil {
		writeError(w, r, err)
		return
	}

	registered, err := s.RegisterZone.Execute(r.Context(), chi.URLParam(r, "siteCode"), req.AreaCode, req.ZoneCode, temperatureClass, req.Hazmat)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/zones/"+registered.ID())
	writeJSON(w, http.StatusCreated, toZoneResponse(registered))
}

func (s *Server) handleListZones(w http.ResponseWriter, r *http.Request) {
	zones, err := s.ListZones.Execute(r.Context(), chi.URLParam(r, "siteCode"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]zoneResponse, 0, len(zones))
	for _, z := range zones {
		out = append(out, toZoneResponse(z))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetZone(w http.ResponseWriter, r *http.Request) {
	found, err := s.GetZone.Execute(r.Context(), chi.URLParam(r, "zoneId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toZoneResponse(found))
}

// --------------------------------------------------------------- aisles ----

func (s *Server) handleRegisterAisle(w http.ResponseWriter, r *http.Request) {
	var req registerAisleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	direction, err := shared.ParseDirection(req.Direction)
	if err != nil {
		writeError(w, r, err)
		return
	}

	zoneID := chi.URLParam(r, "zoneId")
	registered, err := s.RegisterAisle.Execute(r.Context(), zoneID, req.AisleCode, req.SequenceHint, direction)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/zones/"+zoneID+"/aisles/"+registered.AisleCode())
	writeJSON(w, http.StatusCreated, toAisleResponse(registered))
}

func (s *Server) handleListAisles(w http.ResponseWriter, r *http.Request) {
	aisles, err := s.ListAisles.Execute(r.Context(), chi.URLParam(r, "zoneId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]aisleResponse, 0, len(aisles))
	for _, a := range aisles {
		out = append(out, toAisleResponse(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetAisle(w http.ResponseWriter, r *http.Request) {
	aisleID := chi.URLParam(r, "zoneId") + "-" + chi.URLParam(r, "aisleCode")
	found, err := s.GetAisle.Execute(r.Context(), aisleID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAisleResponse(found))
}

// -------------------------------------------------------- location types ---

func (s *Server) handleRegisterLocationType(w http.ResponseWriter, r *http.Request) {
	var req registerLocationTypeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	capacity, err := shared.NewCapacity(req.DefaultCapacity.MaxWeightKg, req.DefaultCapacity.MaxVolumeM3)
	if err != nil {
		writeError(w, r, err)
		return
	}

	registered, err := s.RegisterLocationType.Execute(r.Context(), req.Name, capacity)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/location-types/"+registered.Name())
	writeJSON(w, http.StatusCreated, toLocationTypeResponse(registered))
}

func (s *Server) handleListLocationTypes(w http.ResponseWriter, r *http.Request) {
	types, err := s.ListLocationTypes.Execute(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]locationTypeResponse, 0, len(types))
	for _, t := range types {
		out = append(out, toLocationTypeResponse(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetLocationType(w http.ResponseWriter, r *http.Request) {
	found, err := s.GetLocationType.Execute(r.Context(), chi.URLParam(r, "name"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toLocationTypeResponse(found))
}

// -------------------------------------------------------- placement rules --

func (s *Server) handleDefinePlacementRule(w http.ResponseWriter, r *http.Request) {
	var req definePlacementRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	effect, err := placement.ParseEffect(req.Effect)
	if err != nil {
		writeError(w, r, err)
		return
	}
	predicate, err := placement.NewZonePredicate(req.Zone.ZoneCode, shared.TemperatureClass(req.Zone.TemperatureClass), req.Zone.Hazmat)
	if err != nil {
		writeError(w, r, err)
		return
	}

	defined, err := s.DefinePlacementRule.Execute(r.Context(), req.RuleID, req.LocationType, effect, predicate)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/placement-rules/"+defined.ID())
	writeJSON(w, http.StatusCreated, toPlacementRuleResponse(defined))
}

func (s *Server) handleListPlacementRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.ListPlacementRules.Execute(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]placementRuleResponse, 0, len(rules))
	for _, rule := range rules {
		out = append(out, toPlacementRuleResponse(rule))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetPlacementRule(w http.ResponseWriter, r *http.Request) {
	found, err := s.GetPlacementRule.Execute(r.Context(), chi.URLParam(r, "ruleId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlacementRuleResponse(found))
}

// ------------------------------------------------------- location slots ----

func (s *Server) handleRegisterLocationSlot(w http.ResponseWriter, r *http.Request) {
	var req registerLocationSlotRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	code, err := shared.ParseLocationCode(req.LocationCode)
	if err != nil {
		writeError(w, r, err)
		return
	}

	override := shared.Capacity{}
	if req.CapacityOverride != nil {
		override, err = shared.NewCapacity(req.CapacityOverride.MaxWeightKg, req.CapacityOverride.MaxVolumeM3)
		if err != nil {
			writeError(w, r, err)
			return
		}
	}

	registered, err := s.RegisterLocationSlot.Execute(r.Context(), code, req.LocationType, override)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/locations/"+registered.Code().String())
	writeJSON(w, http.StatusCreated, toLocationSlotResponse(registered))
}

func (s *Server) handleGetLocationSlot(w http.ResponseWriter, r *http.Request) {
	code, err := shared.ParseLocationCode(chi.URLParam(r, "locationCode"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	found, err := s.GetLocationSlot.Execute(r.Context(), code)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toLocationSlotResponse(found))
}

func (s *Server) handleDecommissionLocationSlot(w http.ResponseWriter, r *http.Request) {
	code, err := shared.ParseLocationCode(chi.URLParam(r, "locationCode"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := s.DecommissionLocationSlot.Execute(r.Context(), code); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleImportFacilityLayout answers 200 OK, not 201: a bulk import is a
// partial-success report over many rows, not the creation of one
// addressable resource, so there is no single Location to hand back. The
// per-row outcome is in the body.
func (s *Server) handleImportFacilityLayout(w http.ResponseWriter, r *http.Request) {
	var req []importRowRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	rows := make([]usecases.ImportRow, 0, len(req))
	for _, row := range req {
		rows = append(rows, toImportRow(row))
	}

	report, err := s.ImportFacilityLayout.Execute(r.Context(), rows)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toImportReportResponse(report))
}

// --------------------------------------------- "draw the warehouse" reads --

func (s *Server) handleGetSiteLayout(w http.ResponseWriter, r *http.Request) {
	layout, err := s.GetSiteLayout.Execute(r.Context(), chi.URLParam(r, "siteCode"))
	if err != nil {
		writeError(w, r, err)
		return
	}

	if r.URL.Query().Get("format") == "svg" {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderLayoutSVG(layout)))
		return
	}

	writeJSON(w, http.StatusOK, toSiteLayoutResponse(layout))
}

func (s *Server) handleGetZoneGrid(w http.ResponseWriter, r *http.Request) {
	grid, err := s.GetZoneGrid.Execute(r.Context(), chi.URLParam(r, "zoneId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toZoneGridResponse(grid))
}

// --------------------------------------------------------------- writing ---

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeProblem(w, http.StatusBadRequest, problemInfo{"malformed-request-body", "The request body is not valid JSON"}, err.Error(), r.URL.Path)
		return false
	}
	return true
}

// writeError writes a domain/application error as an RFC 7807
// (application/problem+json) response.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	writeProblem(w, statusFor(err), problemFor(err), err.Error(), r.URL.Path)
}

func writeProblem(w http.ResponseWriter, status int, info problemInfo, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemDetails{
		Type:     problemBaseURI + info.slug,
		Title:    info.title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
