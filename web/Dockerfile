# syntax=docker/dockerfile:1.7
#
# facility_mfe runtime image: a static Module Federation remote served by
# nginx. Kong never serves these bytes -- the fleet's localhost topology puts
# every frontend asset behind the Nginx web gateway on host :80, while Kong
# owns APIs on host :8000 (see warehouse-infra's ADR on the split edges).
#
# @warehouse/ui-kit is a sibling checkout (file:../../warehouse-ui-kit in
# package.json), not a registry package, so it must be supplied as a named
# BuildKit context. Build from this directory:
#
#   docker build --build-context uikit=../../warehouse-ui-kit \
#     -t warehouse/facility-layout-frontend:local .

# Builder base: node:22-bookworm-slim, not node:26-alpine. These
# package-lock.json files were generated on Debian/glibc originally and
# carry linux-x64-gnu/arm64-gnu entries for the platform-specific optional
# dependencies Vite 8 pulls in (@rolldown/binding-linux-*,
# lightningcss-linux-*, @oxlint/binding-linux-*); Debian/glibc matches the
# *-gnu bindings that are actually resolved. Both stages below install with
# `npm ci` against the committed lockfile, which is hash-verified and
# reproducible (unlike an unpinned `npm install`), and was confirmed to
# resolve and build correctly against this Dockerfile's own base image
# (linux/amd64 and linux/arm64) before switching over from the previous
# `npm install --no-save` workaround.
# --- ui-kit build stage (sibling dependency) ---
FROM node:22-bookworm-slim@sha256:83f487e0a63425e5b4d146fb5e5be574bcbe1b7b843d3ebafdd95eaf7767a7e5 AS uikit-build
WORKDIR /uikit
COPY --from=uikit . .
# The uikit named context is a sibling CHECKOUT on the developer's machine
# and may carry its own node_modules from a different host platform, so
# strip it before resolving; the checked-in package-lock.json itself is
# reused via `npm ci`, not regenerated. warehouse-ui-kit is gaining its own
# .dockerignore, but this stays so the build does not depend on which
# revision is checked out.
RUN --mount=type=cache,target=/root/.npm \
    rm -rf node_modules && npm ci && npm run build

# --- remote build stage ---
FROM node:22-bookworm-slim@sha256:83f487e0a63425e5b4d146fb5e5be574bcbe1b7b843d3ebafdd95eaf7767a7e5 AS build
WORKDIR /workspace/warehouse-ui-kit
COPY --from=uikit-build /uikit/package.json ./package.json
COPY --from=uikit-build /uikit/dist ./dist

WORKDIR /workspace/facility-layout/web
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci
COPY . .
RUN npm run build

# --- runtime stage ---
FROM nginxinc/nginx-unprivileged:1.31-alpine@sha256:19c132c9ab02d3b783f478743dafc7a7f42e27aa7d2bdcbec1bb1128ca8f2a07
# apk upgrade picks up CVE fixes published to the alpine branch since the base
# image was last rebuilt, matching warehouse-console's runtime stage. Root is
# only needed for this one step.
USER root
RUN apk upgrade --no-cache
USER nginx
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build --chown=nginx:nginx /workspace/facility-layout/web/dist /usr/share/nginx/html
EXPOSE 8080
