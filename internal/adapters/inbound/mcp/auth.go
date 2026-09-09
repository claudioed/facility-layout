package mcp

import (
	"github.com/claudioed/facility-layout/internal/adapters/inbound/auth"
)

// The static-bearer-key identity model (ADR-0007) now lives in the shared
// inbound auth package (ADR-0014, fleet ADR warehouse-ops-agent 0005) so the
// REST and MCP surfaces of this repository share ONE implementation. These
// aliases keep the MCP adapter's public seam (Scope, Authenticator) stable for
// its composition root and tests; there is no second copy of the logic here.
//
// facility-layout is a read-only Open Host Service, so today it registers only
// read tools and every one requires ScopeRead. The read-write class is kept so
// a future write tool needs no auth rework — just a registration requiring
// ScopeReadWrite.
type (
	// Scope is a coarse authorization class carried by an API key.
	Scope = auth.Scope
	// Authenticator validates a request's bearer credential and reports the
	// scope it grants (StaticKeyAuth today, an OAuth 2.1 resource server
	// tomorrow, behind the same seam).
	Authenticator = auth.Authenticator
)

const (
	ScopeRead      = auth.ScopeRead
	ScopeReadWrite = auth.ScopeReadWrite
)
