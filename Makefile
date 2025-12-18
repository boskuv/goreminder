# Project variables
APP_NAME=goreminder
DOCKER_COMPOSE=docker-compose.yml
CONFIG_FILENAME=config.yaml
CONFIG_FILEPATH=cmd/core/$(CONFIG_FILENAME)
BUILD_DIR=bin
BINARY=bin/$(APP_NAME)

# Go variables
GO=go
GOFLAGS=-mod=vendor
GOTEST_FLAGS=-cover -v
MAIN=cmd/core/main.go

# Version variables
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo "")

# Docker variables
DOCKER=docker
COMPOSE=docker-compose
POSTGRES_CONTAINER=postgres_container
PG_PORT=5432

.PHONY: all lint build run test swagger docker-up docker-down clean version

# Default target
all: build

# Run golangci-lint
lint:
	@golangci-lint run ./...

# Build the Go application
build:
	@echo "Building version $(VERSION)..."
	$(GO) build \
		-ldflags "\
			-X github.com/boskuv/goreminder/pkg/version.Version=$(VERSION) \
			-X github.com/boskuv/goreminder/pkg/version.BuildTime=$(BUILD_TIME) \
			-X github.com/boskuv/goreminder/pkg/version.GitCommit=$(GIT_COMMIT) \
			-X github.com/boskuv/goreminder/pkg/version.GitTag=$(GIT_TAG)" \
		-o $(BINARY) $(MAIN)
	@echo "Copying the configuration file..."
	cp $(CONFIG_FILEPATH) $(BUILD_DIR)/

# Run the application locally
run: build
	./$(BINARY) --configpath $(BUILD_DIR)/$(CONFIG_FILENAME)

# Run tests with coverage
test:
	$(GO) test ./... $(GOTEST_FLAGS)

# Show coverage
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

# Generate Swagger documentation
swagger:
	swag init --dir ./cmd/core,./internal/api/handlers,./internal/models --output ./docs

# Start Docker containers
docker-up:
	$(COMPOSE) -f $(DOCKER_COMPOSE) up -d

# Stop Docker containers
docker-down:
	$(COMPOSE) -f $(DOCKER_COMPOSE) down

# Check database connectivity
db-check:
	$(DOCKER) exec $(POSTGRES_CONTAINER) pg_isready -U postgres -h localhost -p $(PG_PORT)

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Git tag: $(GIT_TAG)"

# Clean the build output
clean:
	rm -rf $(BINARY)
	rm -rf ./bin
