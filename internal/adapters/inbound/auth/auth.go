// Package auth is the fleet-standard REST identity middleware: static bearer
// keys mapped to coarse scopes, no IdP. It is the same posture, and the same
// Authenticator seam, the MCP inbound adapter has carried since ADR-0008 —
// lifted out of internal/adapters/inbound/mcp so both surfaces share one
// implementation per repository (copy-not-share across repositories, per
// the fleet's no-shared-code rule).
//
// Mode is selected with AUTH_MODE:
//
//	enforce  reject unauthenticated/under-scoped requests (401/403, RFC 7807)
//	log      let every request through but log "auth: would-reject" so a
//	         cluster can be rolled out and watched before enforcing
//	off      middleware is a no-op (local dev without keys)
//
// The composition root picks the default: "enforce" when at least one key
// is configured, "off" (with a loud WARN) when none is.
package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// Scope is a coarse authorization class carried by an API key.
type Scope string

const (
	ScopeRead      Scope = "read"
	ScopeReadWrite Scope = "read-write"
)

// Mode selects what the middleware does with a failed authentication.
type Mode string

const (
	ModeEnforce Mode = "enforce"
	ModeLog     Mode = "log"
	ModeOff     Mode = "off"
)

// ParseMode maps the AUTH_MODE env value to a Mode, defaulting to fallback
// on an empty or unknown value.
func ParseMode(raw string, fallback Mode) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case ModeEnforce:
		return ModeEnforce
	case ModeLog:
		return ModeLog
	case ModeOff:
		return ModeOff
	default:
		return fallback
	}
}

// Authenticator validates a request's bearer credential and reports the scope
// it grants. Deliberately an interface: StaticKeyAuth today, an OAuth 2.1
// resource-server implementation tomorrow, behind the same seam.
type Authenticator interface {
	Authenticate(r *http.Request) (scope Scope, ok bool)
}

// StaticKeyAuth authenticates against a fixed set of bearer keys, each
// mapped to a scope. Keys come from a Kubernetes Secret via the composition
// root; this type never logs them. Comparison is constant-time.
type StaticKeyAuth struct {
	keys map[string]Scope
}

// NewStaticKeyAuth builds a StaticKeyAuth from token->scope pairs. Empty
// tokens are ignored so a blank env var cannot silently authorize everyone.
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

// HasKeys reports whether any key is configured — the composition root uses
// it to choose the default Mode.
func (a *StaticKeyAuth) HasKeys() bool { return len(a.keys) > 0 }

// Authenticate extracts a Bearer token and resolves its scope with a
// constant-time comparison against each known key.
func (a *StaticKeyAuth) Authenticate(r *http.Request) (Scope, bool) {
	token, ok := BearerToken(r)
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

// BearerToken pulls the token out of "Authorization: Bearer ***", returning
// ok=false when the header is absent or malformed.
func BearerToken(r *http.Request) (string, bool) {
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

// Allows reports whether a granted scope satisfies a required one.
// read-write satisfies everything; read satisfies only read.
func Allows(granted, required Scope) bool {
	if granted == ScopeReadWrite {
		return true
	}
	return granted == required
}

// RequiredFor is the fleet's default route policy: safe methods need read,
// everything else needs read-write.
func RequiredFor(method string) Scope {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return ScopeRead
	default:
		return ScopeReadWrite
	}
}

type scopeKey struct{}

// ScopeFrom returns the scope the middleware granted to this request, or ""
// when the request was not authenticated (mode off/log).
func ScopeFrom(ctx context.Context) Scope {
	s, _ := ctx.Value(scopeKey{}).(Scope)
	return s
}

// Middleware returns a chi/net-http middleware applying the fleet policy:
// the scope required for a request is RequiredFor(method) unless the caller
// supplies a different policy function. Problem-details bodies follow the
// fleet's RFC 7807 convention with the service's own problem-type base URL.
type Middleware struct {
	Authn       Authenticator
	Mode        Mode
	Logger      *slog.Logger
	ProblemBase string // e.g. "https://errors.<service>.warehouse-systems.dev/"
	// Required overrides the scope policy; nil means RequiredFor(method).
	Required func(r *http.Request) Scope
}

// Handler wraps next.
func (m Middleware) Handler(next http.Handler) http.Handler {
	if m.Mode == ModeOff {
		return next
	}
	logger := m.Logger
	if logger == nil {
		logger = slog.Default()
	}
	required := m.Required
	if required == nil {
		required = func(r *http.Request) Scope { return RequiredFor(r.Method) }
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		granted, ok := m.Authn.Authenticate(r)
		need := required(r)
		switch {
		case !ok:
			if m.Mode == ModeLog {
				logger.WarnContext(r.Context(), "auth: would-reject", "reason", "unauthenticated", "method", r.Method, "path", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("WWW-Authenticate", `Bearer realm="warehouse-systems"`)
			m.problem(w, r, http.StatusUnauthorized, "unauthenticated", "Missing or invalid bearer credential.")
			return
		case !Allows(granted, need):
			if m.Mode == ModeLog {
				logger.WarnContext(r.Context(), "auth: would-reject", "reason", "insufficient-scope", "granted", string(granted), "required", string(need), "method", r.Method, "path", r.URL.Path)
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopeKey{}, granted)))
				return
			}
			m.problem(w, r, http.StatusForbidden, "insufficient-scope", "This credential does not grant the "+string(need)+" scope.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopeKey{}, granted)))
	})
}

func (m Middleware) problem(w http.ResponseWriter, r *http.Request, status int, slug, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":     m.ProblemBase + slug,
		"title":    http.StatusText(status),
		"status":   status,
		"detail":   detail,
		"instance": r.URL.Path,
	})
}

// KeysFromEnv builds the token->scope map the fleet convention expects:
// API_READ_KEY / API_READWRITE_KEY, falling back to MCP_READ_KEY /
// MCP_READWRITE_KEY so one Secret can serve both surfaces.
func KeysFromEnv(getenv func(string) string) map[string]Scope {
	pick := func(primary, fallback string) string {
		if v := getenv(primary); v != "" {
			return v
		}
		return getenv(fallback)
	}
	keys := map[string]Scope{}
	if k := pick("API_READ_KEY", "MCP_READ_KEY"); k != "" {
		keys[k] = ScopeRead
	}
	if k := pick("API_READWRITE_KEY", "MCP_READWRITE_KEY"); k != "" {
		keys[k] = ScopeReadWrite
	}
	return keys
}
