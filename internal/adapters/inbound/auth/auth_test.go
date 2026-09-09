package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newAuthn() *StaticKeyAuth {
	return NewStaticKeyAuth(map[string]Scope{"r-key": ScopeRead, "rw-key": ScopeReadWrite, "": ScopeReadWrite})
}

func do(t *testing.T, m Middleware, method, token string) *httptest.ResponseRecorder {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Scope", string(ScopeFrom(r.Context())))
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(method, "/things", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	m.Handler(next).ServeHTTP(rec, req)
	return rec
}

func TestEnforce_Table(t *testing.T) {
	m := Middleware{Authn: newAuthn(), Mode: ModeEnforce, ProblemBase: "https://errors.test/"}
	cases := []struct {
		name   string
		method string
		token  string
		want   int
		scope  string
	}{
		{"no token GET", http.MethodGet, "", 401, ""},
		{"bad token GET", http.MethodGet, "nope", 401, ""},
		{"read key GET", http.MethodGet, "r-key", 204, "read"},
		{"read key POST", http.MethodPost, "r-key", 403, ""},
		{"rw key POST", http.MethodPost, "rw-key", 204, "read-write"},
		{"rw key GET", http.MethodGet, "rw-key", 204, "read-write"},
		{"empty key never matches", http.MethodGet, "", 401, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(t, m, c.method, c.token)
			if rec.Code != c.want {
				t.Fatalf("want %d got %d body=%s", c.want, rec.Code, rec.Body.String())
			}
			if rec.Code == 204 && rec.Header().Get("X-Scope") != c.scope {
				t.Fatalf("want scope %q in ctx, got %q", c.scope, rec.Header().Get("X-Scope"))
			}
			if rec.Code == 401 && rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("401 must carry WWW-Authenticate")
			}
			if rec.Code >= 400 {
				if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
					t.Fatalf("want problem+json, got %q", ct)
				}
				var body map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("body not json: %v", err)
				}
				if body["status"] != float64(c.want) || body["instance"] != "/things" {
					t.Fatalf("problem body mismatch: %v", body)
				}
			}
		})
	}
}

func TestLogMode_LetsEverythingThroughButLogs(t *testing.T) {
	var logged bool
	logger := slog.New(slog.NewTextHandler(writerFunc(func(p []byte) (int, error) { logged = true; return len(p), nil }), nil))
	m := Middleware{Authn: newAuthn(), Mode: ModeLog, Logger: logger}
	if rec := do(t, m, http.MethodPost, ""); rec.Code != 204 {
		t.Fatalf("log mode must pass unauthenticated requests, got %d", rec.Code)
	}
	if !logged {
		t.Fatal("log mode must log the would-reject")
	}
	logged = false
	if rec := do(t, m, http.MethodPost, "r-key"); rec.Code != 204 || !logged {
		t.Fatalf("log mode must pass under-scoped requests and log, got %d logged=%v", rec.Code, logged)
	}
	logged = false
	if rec := do(t, m, http.MethodGet, "r-key"); rec.Code != 204 || logged {
		t.Fatalf("a valid request must not log a would-reject, got %d logged=%v", rec.Code, logged)
	}
}

func TestOffMode_IsNoop(t *testing.T) {
	m := Middleware{Authn: newAuthn(), Mode: ModeOff}
	if rec := do(t, m, http.MethodDelete, ""); rec.Code != 204 {
		t.Fatalf("off mode must be a no-op, got %d", rec.Code)
	}
}

func TestParseMode(t *testing.T) {
	if ParseMode("ENFORCE", ModeOff) != ModeEnforce || ParseMode(" log ", ModeOff) != ModeLog || ParseMode("", ModeOff) != ModeOff || ParseMode("bogus", ModeEnforce) != ModeEnforce {
		t.Fatal("ParseMode mapping wrong")
	}
}

func TestKeysFromEnv_PrefersAPIThenMCP(t *testing.T) {
	env := map[string]string{"MCP_READ_KEY": "m-r", "API_READWRITE_KEY": "a-rw", "MCP_READWRITE_KEY": "m-rw"}
	keys := KeysFromEnv(func(k string) string { return env[k] })
	if keys["m-r"] != ScopeRead || keys["a-rw"] != ScopeReadWrite || len(keys) != 2 {
		t.Fatalf("unexpected keys %v", keys)
	}
	if NewStaticKeyAuth(KeysFromEnv(func(string) string { return "" })).HasKeys() {
		t.Fatal("no env => no keys")
	}
}

func TestBearerToken_Malformed(t *testing.T) {
	for _, h := range []string{"", "Bearer", "Bearer ", "Basic abc", "bearer   "} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if h != "" {
			r.Header.Set("Authorization", h)
		}
		if _, ok := BearerToken(r); ok {
			t.Fatalf("%q must not parse", h)
		}
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "bearer tok")
	if tok, ok := BearerToken(r); !ok || tok != "tok" {
		t.Fatal("case-insensitive scheme must parse")
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
