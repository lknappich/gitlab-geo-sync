# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
ARG TARGETPLATFORM
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -ldflags "-s -w \
    -X github.com/lknappich/gitlab-geo-sync/internal/version.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev) \
    -X github.com/lknappich/gitlab-geo-sync/internal/version.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
    -X github.com/lknappich/gitlab-geo-sync/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /out/geoctl ./cmd/geoctl

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/geoctl /usr/local/bin/geoctl
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/geoctl"]