BINARY := bin/server
MAIN   := ./cmd/server
PORT   ?= 3020

.PHONY: run start build clean check test migrate migrate-up migrate-down docker-build docker-build-migrate

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

test:
	go test ./...

migrate migrate-up:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; go run ./cmd/migrate up

migrate-down:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; go run ./cmd/migrate down

docker-build:
	docker build --target server -t $(IMAGE) .

docker-build-migrate:
	docker build --target migrate -t $(IMAGE)-migrate .
