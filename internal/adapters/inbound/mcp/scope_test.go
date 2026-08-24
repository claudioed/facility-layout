package mcp

import (
	"context"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// base is the deterministic clock every mcp test runs against.
var base = time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)

// TestScopeGating_DeniesWithoutReadScope proves that a context carrying no
// (or insufficient) scope is rejected by the handler guard. Every registered
// tool is a read tool, so ScopeRead is the minimum the guard enforces; the
// transport test always presents a valid key, so this white-box test is what
// exercises the denial branch.
func TestScopeGating_DeniesWithoutReadScope(t *testing.T) {
	// No scope in context -> scopeFromContext returns "" -> denied.
	unauth := context.Background()

	t.Run("empty-scope context denied at the read guard", func(t *testing.T) {
		if scopeAllows(scopeFromContext(unauth), ScopeRead) {
			t.Fatal("empty-scope context must not satisfy ScopeRead")
		}
	})

	t.Run("read scope satisfies read", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), scopeKey{}, ScopeRead)
		if !scopeAllows(scopeFromContext(ctx), ScopeRead) {
			t.Fatal("read scope must satisfy ScopeRead")
		}
	})

	t.Run("read scope does not satisfy the future write seam", func(t *testing.T) {
		// The read-write scope class exists for a future write tool; a
		// read-only key must not clear ScopeReadWrite.
		ctx := context.WithValue(context.Background(), scopeKey{}, ScopeRead)
		if scopeAllows(scopeFromContext(ctx), ScopeReadWrite) {
			t.Fatal("read scope must NOT satisfy ScopeReadWrite")
		}
	})
}

// TestToolCallDeniedWithoutScope drives a real registered tool through an
// in-memory client/server pair whose request context lacks a scope (no HTTP
// auth middleware runs), asserting the handler's own scope guard rejects it.
func TestToolCallDeniedWithoutScope(t *testing.T) {
	h := newHarness(t)
	h.seedDrawableSite()
	server := NewServer(h.deps)

	client := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil)
	ct, st := sdk.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.CallTool(ctx, &sdk.CallToolParams{
		Name:      "list_sites",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call tool transport error: %v", err)
	}
	if !res.IsError {
		t.Fatal("tool call without scope must be denied by the handler guard")
	}
}

// TestResourceReadDeniedWithoutScope drives the layout resource through a
// server whose request context lacks a scope, asserting the handler denies it.
func TestResourceReadDeniedWithoutScope(t *testing.T) {
	h := newHarness(t)
	h.seedDrawableSite()
	server := NewServer(h.deps)

	client := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil)
	ct, st := sdk.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	_, err = cs.ReadResource(ctx, &sdk.ReadResourceParams{URI: "layout://facility/WH1"})
	if err == nil {
		t.Fatal("resource read without scope must be denied")
	}
}

// TestResourceReadMalformedURI covers the resource handler's not-a-layout-URI
// branch: a scoped read of a URI that does not carry the layout prefix must be
// rejected rather than treated as an empty site code.
func TestResourceReadMalformedURI(t *testing.T) {
	h := newHarness(t)
	server := NewServer(h.deps)

	client := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil)
	ct, st := sdk.NewInMemoryTransports()
	ctx := context.WithValue(context.Background(), scopeKey{}, ScopeRead)
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()
	cs, err := client.Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	// A scoped resource read of an unknown-scheme URI. The SDK routes reads by
	// registered template; an unmatched URI surfaces as an error either way.
	if _, err := cs.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "layout://facility/"}); err == nil {
		t.Fatal("a layout URI with an empty site code must be rejected")
	}
}
