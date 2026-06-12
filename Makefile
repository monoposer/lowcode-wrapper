BINARY := bin/server
MAIN   := ./cmd/server
PORT   ?= 3020

.PHONY: run start build clean check test migrate migrate-up migrate-down docker-build docker-build-cli convert build-convert version version-next version-bump version-set compose-up compose-down postgres-up

CONVERT_BINARY := bin/dataspan-convert
CLI_IMAGE      ?= dataspan-cli:local
IMAGE          ?= dataspan:local

VERSION    := $(shell tr -d '\n' < VERSION 2>/dev/null || echo dev)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w \
	-X github.com/monoposer/dataspan/internal/version.Version=$(VERSION) \
	-X github.com/monoposer/dataspan/internal/version.Commit=$(GIT_COMMIT) \
	-X github.com/monoposer/dataspan/internal/version.BuildDate=$(BUILD_DATE)

run:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; go run $(MAIN)

start: run

build:
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(MAIN)
	@echo "built: $(BINARY)"

clean:
	rm -rf bin/

check:
	go build -ldflags="$(LDFLAGS)" -o /dev/null $(MAIN)
	go build -ldflags="$(LDFLAGS)" -o /dev/null ./cmd/migrate
	go build -ldflags="$(LDFLAGS)" -o /dev/null ./cmd/convert

test:
	go test ./...

migrate migrate-up:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; go run ./cmd/migrate up

migrate-down:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; go run ./cmd/migrate down

docker-build:
	docker build \
		-f deploy/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE) .

docker-build-cli:
	docker build \
		-f deploy/Dockerfile.cli \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(CLI_IMAGE) .

compose-up:
	docker compose -f deploy/docker-compose.yml up -d --build

compose-down:
	docker compose -f deploy/docker-compose.yml down

postgres-up:
	docker compose -f deploy/docker-compose.yml up -d postgres

build-convert convert:
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o $(CONVERT_BINARY) ./cmd/convert
	@echo "built: $(CONVERT_BINARY)"

version:
	@./scripts/version.sh show

version-next:
	@./scripts/version.sh next

BUMP ?= patch

version-bump:
	@./scripts/version.sh bump $(BUMP)

version-set:
	@test -n "$(VER)" || (echo "usage: make version-set VER=1.2.3" >&2; exit 1)
	@./scripts/version.sh set $(VER)
