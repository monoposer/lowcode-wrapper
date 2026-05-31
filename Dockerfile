# syntax=docker/dockerfile:1

# --- build server (static binary, no CGO) ---
FROM golang:1.21-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w" \
    -o /out/server ./cmd/server

# --- build migrate CLI (optional second binary, same image via target) ---
FROM build AS build-migrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w" \
    -o /out/migrate ./cmd/migrate

# --- runtime: distroless static (~2MB) + nonroot user + CA certs for HTTP driver ---
FROM gcr.io/distroless/static-debian12:nonroot AS server
COPY --from=build /out/server /server
EXPOSE 3020
ENTRYPOINT ["/server"]

FROM gcr.io/distroless/static-debian12:nonroot AS migrate
COPY --from=build-migrate /out/migrate /migrate
COPY scripts/migrations /scripts/migrations
ENV MIGRATIONS_DIR=/scripts/migrations
ENTRYPOINT ["/migrate"]
