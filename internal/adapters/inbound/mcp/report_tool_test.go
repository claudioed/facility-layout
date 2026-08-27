package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	inboundmcp "github.com/claudioed/facility-layout/internal/adapters/inbound/mcp"
)

// fakeReportsClient is a test double for the reports REST client the MCP tool
// delegates to.
type fakeReportsClient struct {
	report    inboundmcp.CatalogGrowthReportView
	freshness inboundmcp.FreshnessView
	err       error
	lastQuery inboundmcp.CatalogGrowthQuery
}

func (f *fakeReportsClient) GetCatalogGrowth(_ context.Context, q inboundmcp.CatalogGrowthQuery) (inboundmcp.CatalogGrowthReportView, error) {
	f.lastQuery = q
	return f.report, f.err
}

func (f *fakeReportsClient) GetFreshness(_ context.Context) (inboundmcp.FreshnessView, error) {
	return f.freshness, f.err
}

func TestReportTool_ForwardsFiltersAndReturnsRows(t *testing.T) {
	client := &fakeReportsClient{
		report: inboundmcp.CatalogGrowthReportView{
			Rows: []inboundmcp.CatalogGrowthRowView{
				{Scope: "WH1-STOR-AMB", DayBucket: "2026-06-01T00:00:00Z", SlotsRegistered: 12, AislesRegistered: 3},
			},
		},
	}

	out, err := inboundmcp.GetCatalogGrowthReportForTest(context.Background(), client, inboundmcp.CatalogGrowthToolInput{
		From:        "2026-06-01T00:00:00Z",
		To:          "2026-06-08T00:00:00Z",
		Scope:       "WH1-STOR-AMB",
		Granularity: "day",
	})
	if err != nil {
		t.Fatalf("tool: %v", err)
	}

	if client.lastQuery.From != "2026-06-01T00:00:00Z" || client.lastQuery.Scope != "WH1-STOR-AMB" {
		t.Errorf("filters not forwarded: %+v", client.lastQuery)
	}
	if len(out.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(out.Rows))
	}
	if out.Rows[0].SlotsRegistered != 12 || out.Rows[0].Scope != "WH1-STOR-AMB" {
		t.Errorf("row = %+v", out.Rows[0])
	}
}

func TestReportTool_RequiresFromTo(t *testing.T) {
	client := &fakeReportsClient{}
	tests := []inboundmcp.CatalogGrowthToolInput{
		{To: "2026-06-02T00:00:00Z"},
		{From: "2026-06-01T00:00:00Z"},
	}
	for _, in := range tests {
		if _, err := inboundmcp.GetCatalogGrowthReportForTest(context.Background(), client, in); err == nil {
			t.Errorf("expected error for missing from/to, input=%+v", in)
		}
	}
}

func TestReportTool_NilClient(t *testing.T) {
	if _, err := inboundmcp.GetCatalogGrowthReportForTest(context.Background(), nil, inboundmcp.CatalogGrowthToolInput{From: "a", To: "b"}); err == nil {
		t.Error("expected error when reports client is not configured")
	}
}

// TestReportsRESTClient_CallsEndpoints verifies the real HTTP client hits the
// expected reports paths and decodes the responses.
func TestReportsRESTClient_CallsEndpoints(t *testing.T) {
	var gotPath, gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/reports/catalog-growth":
			_ = json.NewEncoder(w).Encode(inboundmcp.CatalogGrowthReportView{
				Rows: []inboundmcp.CatalogGrowthRowView{{Scope: "WH1", SlotsRegistered: 7}},
			})
		case "/reports/catalog-growth/freshness":
			_ = json.NewEncoder(w).Encode(inboundmcp.FreshnessView{LagSeconds: 12})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c := inboundmcp.NewReportsRESTClient(ts.URL, ts.Client())

	rep, err := c.GetCatalogGrowth(context.Background(), inboundmcp.CatalogGrowthQuery{
		From: "2026-06-01T00:00:00Z", To: "2026-06-08T00:00:00Z", Scope: "WH1", Granularity: "day",
	})
	if err != nil {
		t.Fatalf("GetCatalogGrowth: %v", err)
	}
	if gotPath != "/reports/catalog-growth" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery == "" {
		t.Error("expected query string with filters")
	}
	if len(rep.Rows) != 1 || rep.Rows[0].SlotsRegistered != 7 {
		t.Errorf("report = %+v", rep)
	}

	fr, err := c.GetFreshness(context.Background())
	if err != nil {
		t.Fatalf("GetFreshness: %v", err)
	}
	if fr.LagSeconds != 12 {
		t.Errorf("lag = %v, want 12", fr.LagSeconds)
	}
}

func TestReportsRESTClient_Non2xxIsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := inboundmcp.NewReportsRESTClient(ts.URL, ts.Client())
	if _, err := c.GetCatalogGrowth(context.Background(), inboundmcp.CatalogGrowthQuery{From: "a", To: "b"}); err == nil {
		t.Error("expected error on 500 response")
	}
}
