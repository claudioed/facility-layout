package mcp

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// Scope is a coarse authorization class carried by an API key. Without an IdP
// (ADR-0007), two classes gate the surface: read-only keys may call read
// tools; read-write keys may additionally call write tools.
//
// facility-layout is a read-only Open Host Service, so today it registers only
// read tools and every one requires ScopeRead. The read-write class is kept
// here anyway, identical to the pilot, so the pattern stays uniform across the
// estate and a future write tool needs no auth rework — just a registration
// requiring ScopeReadWrite.
type Scope string

const (
	ScopeRead      Scope = "read"
	ScopeReadWrite Scope = "read-write"
)

// Authenticator validates a request's bearer credential and reports the scope
// it grants. It is deliberately an interface: the current implementation is a
// static bearer key (StaticKeyAuth), and a future OAuth 2.1 resource-server
// implementation can replace it behind this same seam without touching any
// tool handler (ADR-0007, charter §7).
type Authenticator interface {
	// Authenticate returns the scope granted by the request's credentials, or
	// ok=false if the credentials are missing or invalid.
	Authenticate(r *http.Request) (scope Scope, ok bool)
}

// StaticKeyAuth authenticates a request against a fixed set of bearer API
// keys, each mapped to a scope. Keys come from a Kubernetes Secret via the
// composition root; this type never logs them. Comparison is constant-time to
// avoid leaking key material through timing.
type StaticKeyAuth struct {
	// keys maps a bearer token to the scope it grants.
	keys map[string]Scope
}

// NewStaticKeyAuth builds a StaticKeyAuth from token->scope pairs. Empty tokens
// are ignored so a blank env var cannot silently authorize every request.
func NewStaticKeyAuth(keys map[string]Scope) *StaticKeyAuth {
	filtered := make(map[string]Scope, len(keys))
	for k, s := range keys {
		if k == "" {
			continue
		}
		filtered[k] = s
	}
	return &StaticKeyAuth{keys: filtered}
}

// Authenticate extracts a Bearer token from the Authorization header and
// resolves its scope with a constant-time comparison against each known key.
func (a *StaticKeyAuth) Authenticate(r *http.Request) (Scope, bool) {
	token, ok := bearerToken(r)
	if !ok {
		return "", false
	}
	for key, scope := range a.keys {
		if subtle.ConstantTimeCompare([]byte(token), []byte(key)) == 1 {
			return scope, true
		}
	}
	return "", false
}

// bearerToken pulls the token out of an "Authorization: Bearer ***"
// header, returning ok=false when the header is absent or malformed.
func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(h[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

// scopeAllows reports whether a granted scope may call a tool requiring the
// given minimum scope. read-write satisfies everything; read satisfies only
// read.
func scopeAllows(granted, required Scope) bool {
	if granted == ScopeReadWrite {
		return true
	}
	return granted == required
}
