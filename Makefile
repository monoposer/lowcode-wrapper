BINARY := bin/server
MAIN   := ./cmd/server
PORT   ?= 3020

.PHONY: run start build clean check test migrate migrate-up migrate-down docker-build convert build-convert

CONVERT_BINARY := bin/wrapper-convert

IMAGE ?= lowcode-wrapper:local

run:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; go run $(MAIN)

start: run

build:
	@mkdir -p bin
	go build -o $(BINARY) $(MAIN)
	@echo "built: $(BINARY)"

clean:
	rm -rf bin/

check:
	go build -o /dev/null $(MAIN)
	go build -o /dev/null ./cmd/migrate
	go build -o /dev/null ./cmd/convert

test:
	go test ./...

migrate migrate-up:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; go run ./cmd/migrate up

migrate-down:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; go run ./cmd/migrate down

docker-build:
	docker build -t $(IMAGE) .

build-convert convert:
	@mkdir -p bin
	go build -o $(CONVERT_BINARY) ./cmd/convert
	@echo "built: $(CONVERT_BINARY)"
