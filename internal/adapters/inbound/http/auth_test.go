package http_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/claudioed/facility-layout/internal/adapters/inbound/auth"
	inboundhttp "github.com/claudioed/facility-layout/internal/adapters/inbound/http"
)

const (
	testReadKey      = "test-read-key"
	testReadWriteKey = "test-rw-key"
)

// authRouter builds the real OLTP router with the fleet REST identity
// middleware in the given mode, the same way cmd/facility wires it.
func authRouter(mode auth.Mode) http.Handler {
	authn := auth.NewStaticKeyAuth(map[string]auth.Scope{testReadKey: auth.ScopeRead, testReadWriteKey: auth.ScopeReadWrite})
	return inboundhttp.NewRouter(newTestUseCases(), slog.New(slog.NewTextHandler(io.Discard, nil)),
		inboundhttp.WithAuth(auth.Middleware{Authn: authn, Mode: mode}))
}

func authDo(h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	var rd io.Reader = http.NoBody
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, slug string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", ct)
	}
	var problem struct {
		Type     string `json:"type"`
		Status   int    `json:"status"`
		Instance string `json:"instance"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem body is not JSON: %v", err)
	}
	if problem.Type != "https://errors.facility-layout.warehouse-systems.dev/"+slug || problem.Status != status {
		t.Fatalf("problem = %+v, want type slug %q status %d", problem, slug, status)
	}
}

// TestRESTAuth_EnforceTable is the fleet's router table test (ADR-0014): one
// GET and one mutating route, every scope combination, and /healthz open.
func TestRESTAuth_EnforceTable(t *testing.T) {
	h := authRouter(auth.ModeEnforce)
	const siteBody = `{"siteCode":"WH1","name":"Warehouse One"}`

	t.Run("no token on GET -> 401 problem+json with WWW-Authenticate", func(t *testing.T) {
		rec := authDo(h, http.MethodGet, "/sites", "", "")
		assertProblem(t, rec, http.StatusUnauthorized, "unauthenticated")
		if got := rec.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, "Bearer ") {
			t.Fatalf("WWW-Authenticate = %q, want a Bearer challenge", got)
		}
	})

	t.Run("no token on POST -> 401", func(t *testing.T) {
		assertProblem(t, authDo(h, http.MethodPost, "/sites", "", siteBody), http.StatusUnauthorized, "unauthenticated")
	})

	t.Run("bogus token -> 401", func(t *testing.T) {
		assertProblem(t, authDo(h, http.MethodGet, "/sites", "nope", ""), http.StatusUnauthorized, "unauthenticated")
	})

	t.Run("read key on GET -> 200", func(t *testing.T) {
		if rec := authDo(h, http.MethodGet, "/sites", testReadKey, ""); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("read key on POST -> 403 insufficient-scope", func(t *testing.T) {
		assertProblem(t, authDo(h, http.MethodPost, "/sites", testReadKey, siteBody), http.StatusForbidden, "insufficient-scope")
	})

	t.Run("read-write key on POST -> 201", func(t *testing.T) {
		if rec := authDo(h, http.MethodPost, "/sites", testReadWriteKey, siteBody); rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("read-write key on GET -> 200", func(t *testing.T) {
		if rec := authDo(h, http.MethodGet, "/sites/WH1", testReadWriteKey, ""); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("/healthz with no token -> 200", func(t *testing.T) {
		if rec := authDo(h, http.MethodGet, "/healthz", "", ""); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("every mounted group is behind the middleware", func(t *testing.T) {
		for _, path := range []string{"/zones/WH1-STOR-AMB", "/location-types", "/placement-rules", "/locations/WH1-STOR-AMB-A07-03-02-B"} {
			if rec := authDo(h, http.MethodGet, path, "", ""); rec.Code != http.StatusUnauthorized {
				t.Fatalf("GET %s without a token: status = %d, want 401", path, rec.Code)
			}
		}
	})
}

// TestRESTAuth_LogModeLetsRequestsThrough proves the observe-then-enforce
// rollout mode: nothing is rejected, so a cluster can be watched first.
func TestRESTAuth_LogModeLetsRequestsThrough(t *testing.T) {
	h := authRouter(auth.ModeLog)
	if rec := authDo(h, http.MethodGet, "/sites", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("log mode GET without token: status = %d, want 200", rec.Code)
	}
	if rec := authDo(h, http.MethodPost, "/sites", testReadKey, `{"siteCode":"WH1","name":"Warehouse One"}`); rec.Code != http.StatusCreated {
		t.Fatalf("log mode POST with read key: status = %d, want 201", rec.Code)
	}
}

// TestRESTAuth_OffModeIsUnchanged pins the composition-root default when no
// key is configured: the router behaves exactly as before this ADR.
func TestRESTAuth_OffModeIsUnchanged(t *testing.T) {
	h := authRouter(auth.ModeOff)
	if rec := authDo(h, http.MethodPost, "/sites", "", `{"siteCode":"WH1","name":"Warehouse One"}`); rec.Code != http.StatusCreated {
		t.Fatalf("off mode POST without token: status = %d, want 201", rec.Code)
	}
}

// TestReportsAuth_ReadScopeTable covers the reader's router: every /reports
// route needs the READ scope (a read key suffices), /healthz stays open.
func TestReportsAuth_ReadScopeTable(t *testing.T) {
	store := &fakeReportStore{}
	authn := auth.NewStaticKeyAuth(map[string]auth.Scope{testReadKey: auth.ScopeRead, testReadWriteKey: auth.ScopeReadWrite})
	h := inboundhttp.NewReportsRouter(&inboundhttp.ReportsHandlers{Store: store}, nil,
		inboundhttp.WithAuth(auth.Middleware{Authn: authn, Mode: auth.ModeEnforce}))

	const path = "/reports/catalog-growth?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z"

	t.Run("no token -> 401 with challenge", func(t *testing.T) {
		rec := authDo(h, http.MethodGet, path, "", "")
		assertProblem(t, rec, http.StatusUnauthorized, "unauthenticated")
		if got := rec.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, "Bearer ") {
			t.Fatalf("WWW-Authenticate = %q, want a Bearer challenge", got)
		}
	})
	t.Run("read key -> 200", func(t *testing.T) {
		if rec := authDo(h, http.MethodGet, path, testReadKey, ""); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
	})
	t.Run("read-write key -> 200", func(t *testing.T) {
		if rec := authDo(h, http.MethodGet, "/reports/catalog-growth/freshness", testReadWriteKey, ""); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
	})
	t.Run("/healthz with no token -> 200", func(t *testing.T) {
		if rec := authDo(h, http.MethodGet, "/healthz", "", ""); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})
}
