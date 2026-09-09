---
id: 0015-remove-rest-mcp-auth
title: 15. Remove the static-bearer REST/MCP auth layer
sidebar_label: 15. Remove REST/MCP auth
sidebar_position: 15
description: "Reverts ADR-0014 (and the auth portion of ADR-0007): the fleet-standard static bearer key + scope middleware is removed entirely from both the REST and MCP surfaces. Every route and every MCP tool/resource/prompt is now unauthenticated; deployment continues to rely on network boundary (in-cluster only) rather than an application-layer credential."
---

# 15. Remove the static-bearer REST/MCP auth layer

## Status

**Accepted.** 2026-09-09. Undoes the fleet auth rollout recorded in
ADR-0014 (REST) and the auth portion of ADR-0007 (MCP).

## Decision

`internal/adapters/inbound/auth/` — the shared `Scope` / `Authenticator` /
`StaticKeyAuth` / chi `Middleware` implementation — is deleted outright, not
deprecated. Both driving adapters that used it are simplified to have no
authentication concept at all:

- **REST (`internal/adapters/inbound/http`)**: `NewRouter` no longer accepts
  a `WithAuth` option (removed) and no longer wraps any route group in an
  auth middleware; every route — `/sites`, `/zones`, `/location-types`,
  `/placement-rules`, `/locations`, and `/healthz` — is mounted directly,
  exactly as it would be with `AUTH_MODE=off`. `NewReportsRouter` follows the
  same shape for the analytics reader.
- **MCP (`internal/adapters/inbound/mcp`)**: `internal/adapters/inbound/mcp/auth.go`
  is deleted; `Handler(server *mcp.Server)` no longer takes an
  `Authenticator` and no longer checks a bearer credential before dispatching
  to the Streamable HTTP handler. Tool/resource registration no longer
  threads a `scopeOf` function or gates on a required `Scope` — every read
  tool, the `layout://facility/{siteCode}` resource, and the `explore_layout`
  prompt are reachable unconditionally.
- **Composition roots** (`cmd/facility`, `cmd/facility-reports`,
  `cmd/mcp`): the `restAuth(logger) auth.Middleware` helper and all
  `AUTH_MODE`/`API_READ_KEY`/`API_READWRITE_KEY`/`MCP_READ_KEY`/
  `MCP_READWRITE_KEY` environment parsing are removed. These binaries no
  longer read or reference any credential material.
- **Helm chart** (`charts/facility-layout/`): the `auth:` values block, the
  `mcp.readKey`/`mcp.readWriteKey`/`mcp.existingSecret` values,
  `templates/mcp-secret.yaml`, the auth `stringData` entries in
  `templates/secret.yaml`, the `AUTH_MODE` ConfigMap key, and the
  `facility-layout.authSecretName` / `facility-layout.authEnabled` /
  `facility-layout.authEnv` / `facility-layout.mcpSecretName` helpers are all
  removed. `helm lint` and a `helm template` render (main + MCP + analytics
  enabled) are clean with no bearer-key material anywhere in the rendered
  manifests.
- **Docs**: `apis/openapi.yaml` drops the top-level `security:` requirement,
  the per-operation `security: []` override on `/healthz` (now redundant),
  and `components.securitySchemes.bearerAuth`. `README.md` drops the REST
  authentication section, the `AUTH_MODE`/`API_READ_KEY`/`API_READWRITE_KEY`
  env-var rows, and the MCP bearer-key instructions, updating the `curl`/
  `helm` examples to the now-unauthenticated shape. `.spectral.yaml`/CI
  `api-lint` and `helm lint` both pass against the trimmed spec and chart.

This repository's own outbound adapters carry no peer-service API-key logic
(`internal/adapters/outbound/` has no `Authorization`/`Bearer`/`*_API_KEY`
references before or after this change) — facility-layout only ever
authenticated *inbound* callers, so nothing on the outbound side needed
touching.

ADR-0014 and the auth-related passages of ADR-0007 are left in place as
historical record of the decision this reverts; they are not rewritten.

## Consequences

### Easier

- One less package, one less composition-root concern, one less chart
  Secret/ConfigMap surface to keep in sync across nine repositories.
- Local development and every existing handler/tool test needs no key
  scaffolding at all — the removed `auth_test.go` router-table tests and
  `scope_test.go` MCP gating tests are simply gone rather than needing to be
  kept passing in a permanently-`off` mode.

### Harder / accepted

- **No credential enforcement remains on either surface.** Anything able to
  reach the REST port or the MCP port in-cluster can read and mutate the
  warehouse map, and can call every MCP tool, with no application-layer
  gate. This depends entirely on network-level isolation (ClusterIP,
  Kubernetes NetworkPolicy if configured, no public ingress) to keep this
  service's blast radius contained. That boundary is not enforced by this
  repository.
- **Callers that were sending `Authorization: Bearer ...` headers** (e.g. an
  MCP client configured with a `MCP_READ_KEY`, or a REST client configured
  with `API_READ_KEY`) are unaffected functionally — the header is now
  simply ignored — but any client-side code that treated a missing/invalid
  key as a hard requirement should be updated; there is no scope concept to
  configure anymore.
- Re-introducing authentication later (a genuine need re-emerges, or the
  fleet decision is reversed again) means re-adding the middleware from
  scratch or restoring it from this ADR's parent commit — this change does
  not preserve an easy toggle.

## Related

- ADR-0007 — MCP as an inbound adapter (introduced the static-bearer-key
  seam this ADR removes the auth portion of; the adapter shape itself is
  unchanged).
- ADR-0014 — REST identity adoption (the decision this ADR reverts).
- warehouse-ops-agent ADR 0005 — the fleet-wide decision ADR-0014 adopted;
  this repository no longer participates in it.
