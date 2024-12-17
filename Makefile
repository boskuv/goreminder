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

# Docker variables
DOCKER=docker
COMPOSE=docker-compose
POSTGRES_CONTAINER=postgres_container
PG_PORT=5432

.PHONY: all lint build run test swagger docker-up docker-down clean

# Default target
all: build

# Run golangci-lint
lint:
	@golangci-lint run ./...
	
# Build the Go application
build:
	@echo "Building the binary..."
	$(GO) build -o $(BINARY) $(MAIN)
	@echo "Copying the configuration file..."
	cp $(CONFIG_FILEPATH) $(BUILD_DIR)/

# Run the application locally
run: build
	./$(BINARY) --configpath $(BUILD_DIR)/$(CONFIG_FILENAME)

# Run tests with coverage
test:
	$(GO) test ./... $(GOTEST_FLAGS)

# Generate Swagger documentation
swagger:
	swag init ---dir ./cmd/core,./internal/api/handlers,./internal/models --output=./docs

# Start Docker containers
docker-up:
	$(COMPOSE) -f $(DOCKER_COMPOSE) up -d

# Stop Docker containers
docker-down:
	$(COMPOSE) -f $(DOCKER_COMPOSE) down

# Check database connectivity
db-check:
	$(DOCKER) exec $(POSTGRES_CONTAINER) pg_isready -U postgres -h localhost -p $(PG_PORT)

# Clean the build output
clean:
	rm -rf $(BINARY)
	rm -rf ./bin
