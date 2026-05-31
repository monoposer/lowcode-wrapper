BINARY := bin/server
MAIN   := ./cmd/server
PORT   ?= 3020

.PHONY: run start build clean check test

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

test:
	go test ./...
