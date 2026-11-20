# ==============================================================================
# Minecraft Network — Build & Operational Tooling
# ==============================================================================

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

BINARY_NAME ?= network
BIN_DIR ?= bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X github.com/kizio806/minecraft-network/internal/networkcli.Version=$(VERSION) \
	-X github.com/kizio806/minecraft-network/internal/networkcli.GitCommit=$(COMMIT) \
	-X github.com/kizio806/minecraft-network/internal/networkcli.BuildDate=$(DATE)

# Auto-detect Go compiler on host; fallback to containerized Go 1.24
GO_HOST := $(shell command -v go 2>/dev/null)
UID := $(shell id -u)
GID := $(shell id -g)

ifeq ($(GO_HOST),)
	DOCKER_RUN := docker run --rm \
		-u $(UID):$(GID) \
		-e GOCACHE=/app/.cache/go-build \
		-e GOPATH=/app/.cache/go \
		-v "$(CURDIR)":/app \
		-w /app \
		golang:1.24
	GO_CMD := $(DOCKER_RUN) go
else
	GO_CMD := go
endif

GOLANGCI_HOST := $(shell command -v golangci-lint 2>/dev/null)
ifeq ($(GOLANGCI_HOST),)
	LINT_CMD := docker run --rm \
		-u $(UID):$(GID) \
		-e HOME=/tmp \
		-e GOCACHE=/app/.cache/go-build \
		-e GOPATH=/app/.cache/go \
		-e GOLANGCI_LINT_CACHE=/app/.cache/golangci-lint \
		-v "$(CURDIR)":/app \
		-w /app \
		golangci/golangci-lint:v1.64.5 golangci-lint
else
	LINT_CMD := golangci-lint
endif

.PHONY: all
all: build

## help: Display available make targets
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## init-dirs: Ensure build cache directories exist
.PHONY: init-dirs
init-dirs:
	@mkdir -p .cache/go .cache/go-build $(BIN_DIR)

## build: Compile native network CLI binary into bin/network
.PHONY: build
build: init-dirs
	$(GO_CMD) build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/network

## build-bot: Compile native discord-bot daemon into bin/discord-bot
.PHONY: build-bot
build-bot: init-dirs
	$(GO_CMD) build -trimpath -ldflags '-s -w' -o $(BIN_DIR)/discord-bot ./cmd/discord-bot


## test: Run Go unit tests
.PHONY: test
test: init-dirs
	$(GO_CMD) test -v ./...

## test-race: Run Go unit tests with race detection
.PHONY: test-race
test-race: init-dirs
	$(GO_CMD) test -v -race ./...

## vet: Run go vet
.PHONY: vet
vet: init-dirs
	$(GO_CMD) vet ./...

## fmt: Format all Go source files
.PHONY: fmt
fmt: init-dirs
	$(GO_CMD) fmt ./...

## lint: Run golangci-lint
.PHONY: lint
lint: init-dirs
	$(LINT_CMD) run ./...

## tidy: Tidy and verify go modules
.PHONY: tidy
tidy: init-dirs
	$(GO_CMD) mod tidy

## doctor: Run network system diagnostics
.PHONY: doctor
doctor: build
	./$(BIN_DIR)/$(BINARY_NAME) doctor

## health: Run container health checks
.PHONY: health
health: build
	./$(BIN_DIR)/$(BINARY_NAME) health

## validate: Validate plugin manifests, compose files and configs
.PHONY: validate
validate: build
	./$(BIN_DIR)/$(BINARY_NAME) plugins validate
	docker compose -f docker-compose.yml config --quiet

## clean: Remove build artifacts and caches
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)/$(BINARY_NAME) .cache/ .network-operation.lock
