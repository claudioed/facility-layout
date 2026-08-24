package mcp

import (
	"net/http"
	"testing"
)

func req(authHeader string) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "http://x/", nil)
	if authHeader != "" {
		r.Header.Set("Authorization", authHeader)
	}
	return r
}

func TestStaticKeyAuth_Authenticate(t *testing.T) {
	auth := NewStaticKeyAuth(map[string]Scope{
		"read-key":  ScopeRead,
		"write-key": ScopeReadWrite,
		"":          ScopeReadWrite, // empty must be dropped
	})

	tests := []struct {
		name      string
		header    string
		wantScope Scope
		wantOK    bool
	}{
		{"valid read key", "Bearer read-key", ScopeRead, true},
		{"valid write key", "Bearer write-key", ScopeReadWrite, true},
		{"case-insensitive scheme", "bearer read-key", ScopeRead, true},
		{"unknown key", "Bearer nope", "", false},
		{"no header", "", "", false},
		{"wrong scheme", "Basic read-key", "", false},
		{"bearer no token", "Bearer ", "", false},
		{"empty token not authorized", "Bearer ", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scope, ok := auth.Authenticate(req(tc.header))
			if ok != tc.wantOK || scope != tc.wantScope {
				t.Fatalf("Authenticate(%q) = (%q, %v), want (%q, %v)", tc.header, scope, ok, tc.wantScope, tc.wantOK)
			}
		})
	}
}

func TestStaticKeyAuth_EmptyKeySetRejectsAll(t *testing.T) {
	auth := NewStaticKeyAuth(map[string]Scope{"": ScopeReadWrite})
	if _, ok := auth.Authenticate(req("Bearer ")); ok {
		t.Fatal("empty key set must reject all requests")
	}
}

func TestScopeAllows(t *testing.T) {
	tests := []struct {
		granted, required Scope
		want              bool
	}{
		{ScopeReadWrite, ScopeRead, true},
		{ScopeReadWrite, ScopeReadWrite, true},
		{ScopeRead, ScopeRead, true},
		{ScopeRead, ScopeReadWrite, false},
		{"", ScopeRead, false},
	}
	for _, tc := range tests {
		if got := scopeAllows(tc.granted, tc.required); got != tc.want {
			t.Errorf("scopeAllows(%q, %q) = %v, want %v", tc.granted, tc.required, got, tc.want)
		}
	}
}
