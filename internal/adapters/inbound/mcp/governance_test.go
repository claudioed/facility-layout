package mcp

import (
	"context"
	"regexp"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// This file is the computational governance gate for the MCP surface
// (Phase 6). It boots the real server, lists its advertised tools over an
// in-process transport, and asserts the estate-wide rules from
// docs/docs/mcp/governance-charter.md. Because it is a plain `go test`, it
// runs inside the existing CI `test`/`arch-test` jobs with no new
// infrastructure — the left-shift equivalent of `make check` for the MCP
// surface. It reuses this package's existing test harness (newHarness) to
// build Deps, so only the harness wiring is context-specific.

// maxTools is the charter's curated-surface budget (charter §2).
const maxTools = 8

// toolNamePattern enforces the charter's tool-naming convention (§3):
// snake_case, verb_noun, intent-level.
var toolNamePattern = regexp.MustCompile(`^[a-z]+(_[a-z]+)+$`)

// governanceTools connects an in-process client to the real server and returns
// the advertised tool set.
func governanceTools(t *testing.T) []*sdk.Tool {
	t.Helper()
	server := NewServer(newHarness(t).deps)

	client := sdk.NewClient(&sdk.Implementation{Name: "governance", Version: "0"}, nil)
	ct, st := sdk.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	return res.Tools
}

// TestGovernance_ToolCountWithinBudget enforces charter §2.
func TestGovernance_ToolCountWithinBudget(t *testing.T) {
	tools := governanceTools(t)
	if len(tools) == 0 {
		t.Fatal("no tools advertised; expected a curated surface")
	}
	if len(tools) > maxTools {
		names := make([]string, 0, len(tools))
		for _, tool := range tools {
			names = append(names, tool.Name)
		}
		t.Fatalf("tool surface has %d tools, over the charter budget of %d: %v", len(tools), maxTools, names)
	}
}

// TestGovernance_ToolNaming enforces charter §3.
func TestGovernance_ToolNaming(t *testing.T) {
	for _, tool := range governanceTools(t) {
		if !toolNamePattern.MatchString(tool.Name) {
			t.Errorf("tool %q violates the naming convention (want snake_case verb_noun)", tool.Name)
		}
	}
}

// TestGovernance_ToolsAreAnnotated enforces charter §4: every tool carries
// annotations, and a write (non-read-only) tool must be flagged destructive.
func TestGovernance_ToolsAreAnnotated(t *testing.T) {
	for _, tool := range governanceTools(t) {
		if tool.Annotations == nil {
			t.Errorf("tool %q has no annotations (charter §4 requires read/destructive intent)", tool.Name)
			continue
		}
		if !tool.Annotations.ReadOnlyHint {
			if tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
				t.Errorf("write tool %q is not annotated destructive (charter §4)", tool.Name)
			}
		}
	}
}

// TestGovernance_ToolsHaveDescriptions enforces charter §4.
func TestGovernance_ToolsHaveDescriptions(t *testing.T) {
	for _, tool := range governanceTools(t) {
		if tool.Description == "" {
			t.Errorf("tool %q has no description (charter §4)", tool.Name)
		}
	}
}
