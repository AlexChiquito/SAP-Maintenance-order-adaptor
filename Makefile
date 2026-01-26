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

# Test simulator with HTTP calls (starts all services automatically)
test-simulator:
	@echo "========================================="
	@echo "Starting SAP Adaptor End-to-End Test"
	@echo "========================================="
	@echo ""
	@echo "Cleaning up any existing services..."
	@-lsof -ti:8081,8080 | xargs kill -9 2>/dev/null || true
	@sleep 1
	@echo "✅ Ports cleared"
	@echo ""
	@echo "Building binaries..."
	@$(GOBUILD) -o bin/$(SIMULATOR_NAME) ./cmd/simulator
	@$(GOBUILD) -o bin/$(BINARY_NAME) ./cmd/server
	@$(GOBUILD) -o bin/test ./cmd/test
	@echo "✅ Binaries built"
	@echo ""
	@echo "Starting simulator on port 8081..."
	@./bin/$(SIMULATOR_NAME) > /tmp/sap-simulator.log 2>&1 & echo $$! > .simulator.pid
	@sleep 2
	@if ps -p `cat .simulator.pid` > /dev/null 2>&1; then \
		echo "✅ Simulator started (PID: `cat .simulator.pid`)"; \
	else \
		echo "❌ Simulator failed to start"; \
		cat /tmp/sap-simulator.log; \
		rm -f .simulator.pid; \
		exit 1; \
	fi
	@echo ""
	@echo "Starting SAP Adaptor on port 8080..."
	@SAP_ADAPTOR_SAP_BASE_URL=http://localhost:8081 \
		SAP_ADAPTOR_SAP_SIMULATOR_MODE=true \
		SAP_ADAPTOR_DIGITAL_TWIN_BASE_URL=http://localhost:8082 \
		./bin/$(BINARY_NAME) > /tmp/sap-adaptor.log 2>&1 & echo $$! > .adaptor.pid
	@sleep 2
	@if ps -p `cat .adaptor.pid` > /dev/null 2>&1; then \
		echo "✅ SAP Adaptor started (PID: `cat .adaptor.pid`)"; \
	else \
		echo "❌ SAP Adaptor failed to start"; \
		cat /tmp/sap-adaptor.log; \
		kill `cat .simulator.pid` 2>/dev/null || true; \
		rm -f .simulator.pid .adaptor.pid; \
		exit 1; \
	fi
	@echo ""
	@echo "Note: Callback listener not started (optional)"
	@echo "      To see Digital Twin notifications, run in another terminal:"
	@echo "      python3 scripts/listen-callback.py"
	@echo ""
	@echo "Running end-to-end test..."
	@echo "========================================="
	@echo ""
	@./bin/test || (echo ""; echo "❌ Test failed"; kill `cat .simulator.pid .adaptor.pid` 2>/dev/null; rm -f .simulator.pid .adaptor.pid; exit 1)
	@echo ""
	@echo "========================================="
	@echo "Stopping services..."
	@kill `cat .simulator.pid .adaptor.pid` 2>/dev/null || true
	@rm -f .simulator.pid .adaptor.pid
	@echo "✅ Services stopped"
	@echo ""
	@echo "Logs available at:"
	@echo "  - /tmp/sap-simulator.log"
	@echo "  - /tmp/sap-adaptor.log"
	@echo ""
	@echo "========================================="
	@echo "✅ Test completed successfully!"
	@echo "========================================="

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
	rm -f bin/test
	rm -f test-simulator
	rm -f .simulator.pid
	rm -f .adaptor.pid
	rm -f /tmp/sap-simulator.log
	rm -f /tmp/sap-adaptor.log

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
