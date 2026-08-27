# syntax=docker/dockerfile:1.7

# --- build stage ---
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache go.mod/go.sum download separately from source so editing source
# code doesn't bust the module-download layer.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# BuildKit cache mounts for the module and build caches speed up repeat
# builds in CI without baking the cache into the image layers.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/facility ./cmd/facility

# --- runtime stage ---
FROM alpine:3.24
# apk upgrade picks up any CVE fixes published to the 3.24 branch since the
# base image was last rebuilt (e.g. openssl point releases); ca-certificates
# is still pinned explicitly for a reproducible, auditable base layer.
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates=20260611-r0 && \
    addgroup -g 1000 -S app && adduser -u 1000 -S app -G app
WORKDIR /app
COPY --from=build --chown=app:app /out/facility ./facility
COPY --from=build --chown=app:app /src/migrations ./migrations
USER 1000
EXPOSE 8080
ENTRYPOINT ["./facility"]
