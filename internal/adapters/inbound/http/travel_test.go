package http_test

import (
	"net/http"
	"testing"
)

// seedThreeAisleZoneHTTP builds WH1/STOR/AMB with three aisles (A07 TwoWay,
// A08 TwoWay, A09 OneWay) and one slot per bay 01..03 on each.
func (ts *testServer) seedThreeAisleZone() {
	ts.t.Helper()
	ts.seedStorageAisle() // WH1, STOR/AMB, A07 (TwoWay), PalletRack
	ts.seedAisle("WH1-STOR-AMB", "A08", 8, "TwoWay")
	ts.seedAisle("WH1-STOR-AMB", "A09", 9, "OneWay")
	for _, aisleCode := range []string{"A07", "A08", "A09"} {
		for _, bay := range []string{"01", "02", "03"} {
			ts.seedSlot("WH1-STOR-AMB-"+aisleCode+"-"+bay+"-01-A", "PalletRack")
		}
	}
}

func TestRegisterCrossAisleHTTP(t *testing.T) {
	t.Run("registers a connection and returns 201", func(t *testing.T) {
		ts := newTestServer(t)
		ts.seedThreeAisleZone()

		resp := ts.do(http.MethodPost, "/zones/WH1-STOR-AMB/cross-aisles", map[string]any{
			"fromAisle": "A07", "toAisle": "A08", "atBay": "02",
		}).assertStatus(t, http.StatusCreated)

		var body struct {
			ZoneID    string `json:"zoneId"`
			FromAisle string `json:"fromAisle"`
			ToAisle   string `json:"toAisle"`
			AtBay     string `json:"atBay"`
			Active    bool   `json:"active"`
		}
		resp.decode(t, &body)
		if body.ZoneID != "WH1-STOR-AMB" || body.FromAisle != "A07" || body.ToAisle != "A08" || body.AtBay != "02" || !body.Active {
			t.Fatalf("unexpected response %+v", body)
		}
	})

	t.Run("404s on an unknown zone", func(t *testing.T) {
		ts := newTestServer(t)
		ts.seedThreeAisleZone()

		ts.do(http.MethodPost, "/zones/WH1-STOR-XXX/cross-aisles", map[string]any{
			"fromAisle": "A07", "toAisle": "A08", "atBay": "02",
		}).assertProblem(t, http.StatusNotFound, "zone-not-found")
	})

	t.Run("422s on an aisle mismatch", func(t *testing.T) {
		ts := newTestServer(t)
		ts.seedThreeAisleZone()

		ts.do(http.MethodPost, "/zones/WH1-STOR-AMB/cross-aisles", map[string]any{
			"fromAisle": "A07", "toAisle": "A99", "atBay": "02",
		}).assertProblem(t, http.StatusUnprocessableEntity, "cross-aisle-aisle-mismatch")
	})

	t.Run("409s on a duplicate connection", func(t *testing.T) {
		ts := newTestServer(t)
		ts.seedThreeAisleZone()

		ts.do(http.MethodPost, "/zones/WH1-STOR-AMB/cross-aisles", map[string]any{
			"fromAisle": "A07", "toAisle": "A08", "atBay": "02",
		}).assertStatus(t, http.StatusCreated)
		ts.do(http.MethodPost, "/zones/WH1-STOR-AMB/cross-aisles", map[string]any{
			"fromAisle": "A08", "toAisle": "A07", "atBay": "02",
		}).assertProblem(t, http.StatusConflict, "duplicate-cross-aisle")
	})
}

func TestGetZoneTravelGraphHTTP(t *testing.T) {
	ts := newTestServer(t)
	ts.seedThreeAisleZone()
	ts.do(http.MethodPost, "/zones/WH1-STOR-AMB/cross-aisles", map[string]any{
		"fromAisle": "A07", "toAisle": "A08", "atBay": "02",
	}).assertStatus(t, http.StatusCreated)

	resp := ts.do(http.MethodGet, "/zones/WH1-STOR-AMB/travel-graph", nil).assertStatus(t, http.StatusOK)

	var body struct {
		Nodes []struct {
			AisleID string `json:"aisleId"`
			Bay     string `json:"bay"`
		} `json:"nodes"`
		Edges []struct {
			MetresM   float64 `json:"metresM"`
			Estimated bool    `json:"estimated"`
		} `json:"edges"`
	}
	resp.decode(t, &body)
	if len(body.Nodes) != 9 {
		t.Fatalf("expected 9 nodes, got %d", len(body.Nodes))
	}
	if len(body.Edges) == 0 {
		t.Fatal("expected at least one edge")
	}
}

func TestEstimateTravelDistanceHTTP(t *testing.T) {
	t.Run("computes a same-aisle distance", func(t *testing.T) {
		ts := newTestServer(t)
		ts.seedThreeAisleZone()

		resp := ts.do(http.MethodGet, "/distance?from=WH1-STOR-AMB-A07-01-01-A&to=WH1-STOR-AMB-A07-03-01-A", nil).
			assertStatus(t, http.StatusOK)

		var body struct {
			MetresM   float64 `json:"metresM"`
			Estimated bool    `json:"estimated"`
			Route     []struct {
				AisleID string `json:"aisleId"`
				Bay     string `json:"bay"`
			} `json:"route"`
		}
		resp.decode(t, &body)
		if body.MetresM != 2.4 {
			t.Fatalf("expected 2.4m (2 gaps at the default 1.2m bay pitch), got %v", body.MetresM)
		}
		if !body.Estimated {
			t.Fatal("expected Estimated=true: no aisle geometry was ever set")
		}
		if len(body.Route) != 3 {
			t.Fatalf("expected a 3-waypoint route, got %d", len(body.Route))
		}
	})

	t.Run("422s on a cross-zone request", func(t *testing.T) {
		ts := newTestServer(t)
		ts.seedThreeAisleZone()
		ts.seedZone("STOR", "FRZ", "Frozen", false)
		ts.seedAisle("WH1-STOR-FRZ", "B01", 1, "TwoWay")
		ts.seedSlot("WH1-STOR-FRZ-B01-01-01-A", "PalletRack")

		ts.do(http.MethodGet, "/distance?from=WH1-STOR-AMB-A07-01-01-A&to=WH1-STOR-FRZ-B01-01-01-A", nil).
			assertProblem(t, http.StatusUnprocessableEntity, "no-route-between-zones")
	})

	t.Run("404s when a location does not exist", func(t *testing.T) {
		ts := newTestServer(t)
		ts.seedThreeAisleZone()

		ts.do(http.MethodGet, "/distance?from=WH1-STOR-AMB-A99-01-01-A&to=WH1-STOR-AMB-A07-01-01-A", nil).
			assertProblem(t, http.StatusNotFound, "location-slot-not-found")
	})

	t.Run("400s on a malformed location code", func(t *testing.T) {
		ts := newTestServer(t)
		ts.seedThreeAisleZone()

		ts.do(http.MethodGet, "/distance?from=not-a-code&to=WH1-STOR-AMB-A07-01-01-A", nil).
			assertProblem(t, http.StatusBadRequest, "malformed-location-code")
	})
}
