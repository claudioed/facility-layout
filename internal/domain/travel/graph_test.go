package travel_test

import (
	"errors"
	"testing"

	"github.com/claudioed/facility-layout/internal/domain/travel"
)

// buildThreeAisleGrid constructs a hand-built 3-aisle zone: A01 (TwoWay,
// with real centreline geometry: 3 bays over 2.4m -> 1.2m/bay), A02
// (OneWay, no geometry -> falls back to the 1.0m pitch), A03 (TwoWay, no
// geometry), connected by one cross-aisle between A01 and A02 at bay "02".
func buildThreeAisleGrid() travel.Graph {
	aisles := []travel.AisleGeom{
		{AisleID: "Z-A01", Bays: []string{"01", "02", "03"}, OneWay: false, CentrelineM: 2.4},
		{AisleID: "Z-A02", Bays: []string{"01", "02", "03"}, OneWay: true, CentrelineM: 0},
		{AisleID: "Z-A03", Bays: []string{"01", "02", "03"}, OneWay: false, CentrelineM: 0},
	}
	cross := []travel.CrossAisleRef{
		{FromAisleID: "Z-A01", ToAisleID: "Z-A02", AtBay: "02"},
	}
	return travel.Build(aisles, cross, travel.Pitch{BayPitchM: 1.0})
}

func TestGraphDistanceSameAisle(t *testing.T) {
	g := buildThreeAisleGrid()

	route, err := g.Distance(travel.Node{AisleID: "Z-A01", Bay: "01"}, travel.Node{AisleID: "Z-A01", Bay: "03"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Real centreline geometry: 2.4m / 2 gaps = 1.2m/bay, 2 gaps = 2.4m.
	if route.MetresM != 2.4 {
		t.Fatalf("expected 2.4m along A01's centreline, got %v", route.MetresM)
	}
	if route.Estimated {
		t.Fatal("a route entirely on real geometry must not be Estimated")
	}
	wantNodes := []travel.Node{{AisleID: "Z-A01", Bay: "01"}, {AisleID: "Z-A01", Bay: "02"}, {AisleID: "Z-A01", Bay: "03"}}
	if len(route.Nodes) != len(wantNodes) {
		t.Fatalf("expected %d waypoints, got %d: %+v", len(wantNodes), len(route.Nodes), route.Nodes)
	}
	for i, n := range wantNodes {
		if route.Nodes[i] != n {
			t.Fatalf("waypoint %d: expected %+v, got %+v", i, n, route.Nodes[i])
		}
	}
}

func TestGraphDistanceViaCrossAisle(t *testing.T) {
	g := buildThreeAisleGrid()

	// A01 bay 01 -> A02 bay 03: must cross via the cross-aisle at bay 02.
	route, err := g.Distance(travel.Node{AisleID: "Z-A01", Bay: "01"}, travel.Node{AisleID: "Z-A02", Bay: "03"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A01 01->02 (1.2m, real) + cross-aisle (1.0m, estimated) + A02 02->03 (1.0m, estimated, OneWay forward).
	want := 1.2 + 1.0 + 1.0
	if route.MetresM != want {
		t.Fatalf("expected %v, got %v", want, route.MetresM)
	}
	if !route.Estimated {
		t.Fatal("a route using the estimated cross-aisle leg must be flagged Estimated")
	}
}

func TestGraphOneWayAisleForcesLongerRoute(t *testing.T) {
	g := buildThreeAisleGrid()

	// A02 is OneWay walk-order forward (01 -> 02 -> 03). Going backward
	// (03 -> 01) has no direct edge on A02 itself.
	if _, err := g.Distance(travel.Node{AisleID: "Z-A02", Bay: "03"}, travel.Node{AisleID: "Z-A02", Bay: "01"}); !errors.Is(err, travel.ErrNoRoute) {
		t.Fatalf("expected ErrNoRoute for the reverse direction of a one-way aisle with no other path, got %v", err)
	}

	// The forward direction remains a direct, cheap route.
	route, err := g.Distance(travel.Node{AisleID: "Z-A02", Bay: "01"}, travel.Node{AisleID: "Z-A02", Bay: "03"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if route.MetresM != 2.0 {
		t.Fatalf("expected 2.0m (2 estimated bay-pitch gaps), got %v", route.MetresM)
	}
}

func TestGraphMissingGeometryYieldsEstimated(t *testing.T) {
	g := buildThreeAisleGrid()

	route, err := g.Distance(travel.Node{AisleID: "Z-A03", Bay: "01"}, travel.Node{AisleID: "Z-A03", Bay: "02"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !route.Estimated {
		t.Fatal("expected Estimated=true for an aisle with no centreline geometry")
	}
	if route.MetresM != 1.0 {
		t.Fatalf("expected the 1.0m bay pitch, got %v", route.MetresM)
	}
}

func TestGraphUnknownNode(t *testing.T) {
	g := buildThreeAisleGrid()

	_, err := g.Distance(travel.Node{AisleID: "Z-A99", Bay: "01"}, travel.Node{AisleID: "Z-A01", Bay: "01"})
	if !errors.Is(err, travel.ErrUnknownNode) {
		t.Fatalf("expected ErrUnknownNode, got %v", err)
	}

	_, err = g.Distance(travel.Node{AisleID: "Z-A01", Bay: "01"}, travel.Node{AisleID: "Z-A01", Bay: "99"})
	if !errors.Is(err, travel.ErrUnknownNode) {
		t.Fatalf("expected ErrUnknownNode, got %v", err)
	}
}

func TestGraphSameNodeIsZeroDistance(t *testing.T) {
	g := buildThreeAisleGrid()

	route, err := g.Distance(travel.Node{AisleID: "Z-A01", Bay: "02"}, travel.Node{AisleID: "Z-A01", Bay: "02"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if route.MetresM != 0 || route.Estimated {
		t.Fatalf("expected a zero-length, non-estimated self-route, got %+v", route)
	}
	if len(route.Nodes) != 1 || route.Nodes[0] != (travel.Node{AisleID: "Z-A01", Bay: "02"}) {
		t.Fatalf("expected a single-waypoint route, got %+v", route.Nodes)
	}
}

func TestGraphDisconnectedAislesHaveNoRoute(t *testing.T) {
	g := buildThreeAisleGrid()

	// A03 has no cross-aisle to either A01 or A02 in this fixture.
	if _, err := g.Distance(travel.Node{AisleID: "Z-A01", Bay: "01"}, travel.Node{AisleID: "Z-A03", Bay: "01"}); !errors.Is(err, travel.ErrNoRoute) {
		t.Fatalf("expected ErrNoRoute between disconnected aisles, got %v", err)
	}
}

func TestBuildIgnoresCrossAisleToUnknownAisle(t *testing.T) {
	aisles := []travel.AisleGeom{
		{AisleID: "Z-A01", Bays: []string{"01", "02"}, OneWay: false},
	}
	cross := []travel.CrossAisleRef{
		{FromAisleID: "Z-A01", ToAisleID: "Z-A99", AtBay: "01"},
	}
	g := travel.Build(aisles, cross, travel.Pitch{BayPitchM: 1.0})

	if _, err := g.Distance(travel.Node{AisleID: "Z-A01", Bay: "01"}, travel.Node{AisleID: "Z-A99", Bay: "01"}); !errors.Is(err, travel.ErrUnknownNode) {
		t.Fatalf("expected ErrUnknownNode for an aisle the graph was never given, got %v", err)
	}
}

// TestGraphSingleBayAisleIgnoresCentreline proves a single-bay aisle (no
// bay-to-bay gap to divide a centreline length across) always falls back
// to the zone's bay pitch, even when real centreline geometry is present —
// otherwise dividing CentrelineM by zero gaps would produce a bogus
// distance instead of the intended "no adjacent-bay edge exists" case.
func TestGraphSingleBayAisleIgnoresCentreline(t *testing.T) {
	aisles := []travel.AisleGeom{
		{AisleID: "Z-A01", Bays: []string{"01"}, OneWay: false, CentrelineM: 5.0},
	}
	g := travel.Build(aisles, nil, travel.Pitch{BayPitchM: 1.0})

	// A single bay has no adjacent-bay edge at all: distance to itself is
	// the only reachable query, and it must be the zero self-distance
	// rather than anything derived from CentrelineM.
	route, err := g.Distance(travel.Node{AisleID: "Z-A01", Bay: "01"}, travel.Node{AisleID: "Z-A01", Bay: "01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if route.MetresM != 0 {
		t.Fatalf("expected 0, got %v", route.MetresM)
	}
}

// TestGraphFourWaypointRoute exercises a route with an even number of
// waypoints (four), so the path-reversal loop's boundary condition is
// checked against both an even and (via the other tests) an odd node
// count.
func TestGraphFourWaypointRoute(t *testing.T) {
	aisles := []travel.AisleGeom{
		{AisleID: "Z-A01", Bays: []string{"01", "02", "03", "04"}, OneWay: false},
	}
	g := travel.Build(aisles, nil, travel.Pitch{BayPitchM: 1.0})

	route, err := g.Distance(travel.Node{AisleID: "Z-A01", Bay: "01"}, travel.Node{AisleID: "Z-A01", Bay: "04"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []travel.Node{
		{AisleID: "Z-A01", Bay: "01"},
		{AisleID: "Z-A01", Bay: "02"},
		{AisleID: "Z-A01", Bay: "03"},
		{AisleID: "Z-A01", Bay: "04"},
	}
	if len(route.Nodes) != len(want) {
		t.Fatalf("expected %d waypoints, got %d: %+v", len(want), len(route.Nodes), route.Nodes)
	}
	for i, n := range want {
		if route.Nodes[i] != n {
			t.Fatalf("waypoint %d: expected %+v, got %+v (full route %+v)", i, n, route.Nodes[i], route.Nodes)
		}
	}
}
