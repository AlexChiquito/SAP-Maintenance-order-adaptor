# SAP Adaptor Makefile

# Variables
BINARY_NAME=sap-adaptor
SIMULATOR_NAME=sap-simulator
GO=go
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOTEST=$(GO) test
GOGET=$(GO) get
GOMOD=$(GO) mod

# Main targets
.PHONY: all build clean test run help

all: build

# Build main adaptor
build:
	$(GOBUILD) -o bin/$(BINARY_NAME) -v ./cmd/server

# Build simulator
build-simulator:
	$(GOBUILD) -o bin/$(SIMULATOR_NAME) -v ./cmd/simulator

# Build both
build-all: build build-simulator

# Run main adaptor
run:
	$(GOBUILD) -o bin/$(BINARY_NAME) -v ./cmd/server
	./bin/$(BINARY_NAME)

# Run simulator
run-simulator:
	$(GO) run ./cmd/simulator

# Test
test:
	$(GOTEST) -v ./...

# Test simulator with HTTP calls
test-simulator:
	$(GOBUILD) -o test-simulator -v ./cmd/test
	./test-simulator
	rm -f test-simulator

# Test with simulator running
test-full:
	@echo "Starting simulator in background..."
	@$(GO) run ./cmd/simulator & echo $$! > .simulator.pid && \
	sleep 2 && \
	echo "Running tests..." && \
	$(GO) run ./cmd/test && \
	kill `cat .simulator.pid` && \
	rm .simulator.pid

# Clean
clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -f bin/$(SIMULATOR_NAME)
	rm -f test-simulator
	rm -f .simulator.pid

# Dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Docker
docker-build:
	docker build --target adaptor -t $(BINARY_NAME) .
	docker build --target simulator -t $(SIMULATOR_NAME) .

docker-run:
	docker compose up -d

docker-stop:
	docker compose down

docker-logs:
	docker compose logs -f

# Help
help:
	@echo "SAP Adaptor Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  build              - Build the main adaptor"
	@echo "  build-simulator    - Build the simulator"
	@echo "  build-all          - Build both adaptor and simulator"
	@echo "  run                - Build and run the main adaptor"
	@echo "  run-simulator      - Run the simulator"
	@echo "  test               - Run unit tests"
	@echo "  test-simulator     - Test simulator functionality"
	@echo "  test-full          - Test with simulator running in background"
	@echo "  clean              - Clean build artifacts"
	@echo "  deps               - Download and tidy dependencies"
	@echo "  docker-build       - Build Docker images"
	@echo "  docker-run         - Run with docker-compose"
	@echo "  docker-stop        - Stop docker-compose"
	@echo "  docker-logs        - View docker logs"
	@echo "  help               - Show this help message"
