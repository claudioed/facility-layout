#!/usr/bin/env bash
set -euo pipefail

chart_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
release=facility-layout

default_render="$(helm template "$release" "$chart_dir")"
enabled_render="$(helm template "$release" "$chart_dir" --set frontend.enabled=true)"
api_deployment="$(helm template "$release" "$chart_dir" --show-only templates/deployment.yaml)"
api_service="$(helm template "$release" "$chart_dir" --show-only templates/service.yaml)"
frontend_deployment="$(helm template "$release" "$chart_dir" --set frontend.enabled=true --show-only templates/frontend-deployment.yaml)"
frontend_service="$(helm template "$release" "$chart_dir" --set frontend.enabled=true --show-only templates/frontend-service.yaml)"

if grep -q 'name: facility-layout-frontend' <<<"$default_render"; then
  echo "frontend resources rendered while frontend.enabled=false" >&2
  exit 1
fi

[[ "$(grep -c 'app.kubernetes.io/component: api' <<<"$api_deployment")" -ge 3 ]]
[[ "$(grep -c 'app.kubernetes.io/component: api' <<<"$api_service")" -ge 2 ]]
[[ "$(grep -c 'app.kubernetes.io/component: frontend' <<<"$frontend_deployment")" -ge 3 ]]
[[ "$(grep -c 'app.kubernetes.io/component: frontend' <<<"$frontend_service")" -ge 2 ]]

grep -q 'type: ClusterIP' <<<"$frontend_service"
if grep -Eq 'kind: (Ingress|HTTPRoute)|type: (NodePort|LoadBalancer)' <<<"$frontend_deployment"$'\n'"$frontend_service"; then
  echo "frontend render exposes a forbidden public resource" >&2
  exit 1
fi

for component in api analytics-projector analytics-reports mcp frontend; do
  grep -q "app.kubernetes.io/component: $component" <<<"$(helm template "$release" "$chart_dir" --set frontend.enabled=true --set analytics.enabled=true --set mcp.enabled=true --set analytics.database.projectorUrl=postgres://projector --set analytics.database.reportsUrl=postgres://reports)"
done

api_selector="$(grep -A4 '^  selector:$' <<<"$api_service")"
frontend_selector="$(grep -A4 '^  selector:$' <<<"$frontend_service")"
grep -q 'app.kubernetes.io/component: api' <<<"$api_selector"
grep -q 'app.kubernetes.io/component: frontend' <<<"$frontend_selector"
[[ "$api_selector" != "$frontend_selector" ]]

grep -q 'name: facility-layout-frontend' <<<"$enabled_render"
echo "facility-layout chart render assertions passed"
