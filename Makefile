BINARY_SERVER  := bin/p2p-server
BINARY_CLI     := bin/p2p-cli
MODULE         := github.com/yourorg/p2p-messenger
DOCKER_IMAGE   := p2p-messenger

.PHONY: all build build-server build-cli run run-cli test test-unit test-integration test-e2e lint clean docker docker-compose proto help

all: build

## Build
build: build-server build-cli

build-server:
	@mkdir -p bin
	go build -ldflags="-w -s" -o $(BINARY_SERVER) ./cmd/server

build-cli:
	@mkdir -p bin
	go build -ldflags="-w -s" -o $(BINARY_CLI) ./cmd/cli

## Run
run: build-server
	./$(BINARY_SERVER)

run-cli: build-cli
	./$(BINARY_CLI)

## Test
test: test-unit test-integration

test-unit:
	go test -v -race ./internal/... ./pkg/...

test-integration:
	go test -v -race -timeout 60s ./test/integration/...

test-e2e:
	go test -v -race -timeout 120s ./test/e2e/...

## Code quality
lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

## Proto — requires protoc and protoc-gen-go
proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--proto_path=api/proto \
		api/proto/message.proto

## Docker
docker:
	docker build -f deployments/docker/Dockerfile -t $(DOCKER_IMAGE):latest .

docker-compose:
	docker compose -f deployments/compose/docker-compose.yaml up --build

## Clean
clean:
	rm -rf bin/
	go clean -testcache

help:
	@grep -E '^[a-zA-Z_-]+:' Makefile | awk -F: '{printf "  %-20s\n", $$1}'
