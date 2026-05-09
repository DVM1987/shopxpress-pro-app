# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.22
ARG DISTROLESS_TAG=nonroot

# ─── Stage 1: build ─────────────────────────────────────────
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build

ARG SERVICE
RUN test -n "${SERVICE}" || (echo "ERROR: --build-arg SERVICE=<name> is required" >&2 && exit 1)

ENV CGO_ENABLED=0 \
    GOFLAGS=-trimpath \
    GOOS=linux

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download -x

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w -buildid=" \
        -o /out/app ./services/${SERVICE}

# ─── Stage 2: runtime (distroless static, ~2 MB) ────────────
FROM gcr.io/distroless/static-debian12:${DISTROLESS_TAG}

ARG SERVICE
LABEL org.opencontainers.image.source="https://github.com/DVM1987/shopxpress-pro-app" \
      org.opencontainers.image.title="shopxpress-pro/${SERVICE}" \
      org.opencontainers.image.licenses="MIT"

COPY --from=build /out/app /app

USER nonroot:nonroot
ENTRYPOINT ["/app"]
