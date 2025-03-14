.PHONY: build run clean test fmt lint docker-build docker-run

# Application name and main package
APP_NAME=setbull_trader
MAIN_PACKAGE=setbull_trader

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
BUILD_FLAGS=-v
LDFLAGS=-ldflags "-X main.Version=$$(git describe --tags --always --dirty) -X main.BuildTime=$$(date +%Y-%m-%dT%H:%M:%S)"

# Docker settings
DOCKER_IMG=$(APP_NAME)
DOCKER_TAG=latest

# Default target
all: fmt test build

# Build the application
build:
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o bin/$(APP_NAME) .

# Run the application
run:
	$(GORUN) $(BUILD_FLAGS) main.go

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f $(APP_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Format code
fmt:
	$(GOFMT) ./...

# Run linter
lint:
	golangci-lint run ./...

# Install dependencies
deps:
	$(GOMOD) download
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMG):$(DOCKER_TAG) .

# Run Docker container
docker-run:
	docker run -p 8080:8080 --name $(APP_NAME) -d $(DOCKER_IMG):$(DOCKER_TAG)

# Initialize a new application.yaml from the example
init-config:
	@if [ ! -f application.yaml ]; then \
		cp application.example.yaml application.yaml; \
		echo "Created application.yaml from example. Please update it with your configuration."; \
	else \
		echo "application.yaml already exists."; \
	fi

# Help
help:
	@echo "Available commands:"
	@echo "  make all          - Format, test, and build"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run the application"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linter"
	@echo "  make deps         - Install dependencies"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"
	@echo "  make init-config  - Initialize configuration file"