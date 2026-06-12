# syntax=docker/dockerfile:1

ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_DATE=unknown

# --- build server (static binary, no CGO) ---
FROM golang:1.25-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src

ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE

# Default GOCACHE (/root/.cache/go-build) and GOTMPDIR (/tmp) can be read-only under
# BuildKit; keep all build artifacts under WORKDIR (writable layer).
ENV CGO_ENABLED=0 \
    GOPROXY=https://proxy.golang.org,direct \
    GOMODCACHE=/go/pkg/mod \
    GOCACHE=/src/.gocache \
    GOTMPDIR=/src/.gotmp

RUN mkdir -p /src/.gocache /src/.gotmp /out

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/src/.gocache \
    GOOS=linux go build -trimpath \
    -ldflags="-s -w \
      -X github.com/monoposer/dataspan/internal/version.Version=${VERSION} \
      -X github.com/monoposer/dataspan/internal/version.Commit=${GIT_COMMIT} \
      -X github.com/monoposer/dataspan/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/server ./cmd/server

# --- runtime: distroless static (~2MB) + nonroot user + CA certs for HTTP driver ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 3020
ENTRYPOINT ["/server"]
