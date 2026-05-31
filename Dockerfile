# syntax=docker/dockerfile:1

# --- build server (static binary, no CGO) ---
FROM golang:1.24-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src

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
    -ldflags="-s -w" \
    -o /out/server ./cmd/server

# --- runtime: distroless static (~2MB) + nonroot user + CA certs for HTTP driver ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 3020
ENTRYPOINT ["/server"]
