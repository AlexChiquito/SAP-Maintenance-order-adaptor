# SAP Adaptor

A Go-based service that acts as a bridge between Digital Twin Event Generator systems and SAP Plant Maintenance.

## Overview

The SAP Adaptor follows the integration workflow specified in the architecture:

1. **Receives Maintenance Order Event** from Digital Twin
2. **Creates SAP Maintenance Notification** 
3. **Creates SAP Maintenance Order** with notification reference
4. **Monitors order status** until completion
5. **Sends Maintenance Done Event** back to Digital Twin

## API Endpoints

### Maintenance Orders
- `POST /api/v1/maintenance-orders` - Create maintenance order event
- `GET /api/v1/maintenance-orders/{id}` - Get maintenance order status

### Maintenance Events  
- `POST /api/v1/maintenance-done` - Handle maintenance completion event

### System
- `GET /health` - Health check
- `GET /metrics` - Service metrics

## Testing

To run the end-to-end test:

```bash
# Terminal 1: Start the simulator
./bin/simulator

# Terminal 2: Start the adaptor (in simulator mode)
SAP_ADAPTOR_SAP_BASE_URL=http://localhost:8081 \
SAP_ADAPTOR_SAP_SIMULATOR_MODE=true \
SAP_ADAPTOR_DIGITAL_TWIN_BASE_URL=http://localhost:8082 \
./bin/sap-adaptor

# Terminal 3: Run the test
make test-simulator
```

The test demonstrates the complete workflow:
1. Digital Twin sends maintenance order event
2. Adaptor creates notification and order in SAP
3. Adaptor polls order status until completion
4. Final order includes components and equipment data

## Quick Start

### Prerequisites
- Docker and Docker Compose (recommended)
- Or Go 1.21 or later (for local development)
- Access to SAP Plant Maintenance system (optional - simulator mode available)
- Digital Twin system (optional for testing)

### Option 1: Using Docker Compose (Recommended)

The project includes a `docker-compose.yml` file for easy deployment.

#### Quick Start with Default Settings
You can start the service immediately with default settings:
```bash
docker compose up -d
```

This will start the service in simulator mode, which is perfect for testing and development. By default:
- The service runs on port 8080
- Simulator mode is enabled (no real SAP connection required)
- Log level is set to "info"
- Health checks are configured
- Automatic restarts are enabled
- Logs are persisted in a Docker volume

#### Custom Configuration
For production or custom settings:

1. Copy the environment file for Docker:
```bash
cp env.docker .env
# Edit .env with your configuration
```

2. Start the service with your custom settings:
```bash
docker compose up -d
```

#### Managing the Service
To stop the service:
```bash
docker compose down
```

View the logs in real-time:
```bash
docker compose logs -f
```
### Option 2: Local Development Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd sap-adaptor
```

2. Choose one of these methods:

#### A. Using the setup script (Recommended)
```bash
./setup.sh
```

#### B. Manual setup
```bash
# Install dependencies
go mod tidy

# Configure environment
cp env.example .env
# Edit .env with your configuration (simulator mode is enabled by default)

# Run the service
go run ./cmd/server
```

The service will start on `http://localhost:8080` in **simulator mode** by default.

## Configuration

The service can be configured via:

1. **Environment variables** 
2. **YAML configuration file** (`config.yaml`)

### Simulator Mode (Default)

For testing and demonstration purposes, the service runs in **simulator mode** by default:


### Production Mode

To connect to a real SAP system:

1. Set `SAP_ADAPTOR_SAP_SIMULATOR_MODE=false`
2. Configure SAP connection details:
   - `SAP_ADAPTOR_SAP_BASE_URL` - SAP API base URL
   - `SAP_ADAPTOR_SAP_CLIENT_ID` - OAuth client ID
   - `SAP_ADAPTOR_SAP_CLIENT_SECRET` - OAuth client secret
   - `SAP_ADAPTOR_SAP_TOKEN_URL` - OAuth token endpoint

### Optional Configuration

- `SAP_ADAPTOR_SERVER_PORT` - Server port (default: 8080)
- `SAP_ADAPTOR_SAP_TIMEOUT` - SAP API timeout in seconds (default: 30)
- `SAP_ADAPTOR_LOG_LEVEL` - Log level (default: info)

## API Documentation

The OpenAPI specification is available at:
- Swagger UI: `http://localhost:8080/swagger/index.html`
- OpenAPI spec: `http://localhost:8080/swagger/doc.json`


### Manual Testing

#### Create Maintenance Order

```bash
curl -X POST http://localhost:8080/api/v1/maintenance-orders \
  -H "Content-Type: application/json" \
  -d '{
    "equipmentId": "10000045",
    "functionalLocation": "FL100-200-300",
    "plant": "1000",
    "description": "Replace pump seal due to leakage",
    "priority": "3",
    "maintenanceOrderType": "PM01",
    "plannedStartTime": "2025-08-21T08:00:00Z",
    "plannedEndTime": "2025-08-21T16:00:00Z",
    "operations": [
      {
        "text": "Disassemble pump",
        "workCenter": "PUMP-WC01",
        "duration": 4,
        "durationUnit": "H"
      }
    ]
  }'
```

**Response (Simulator Mode):**
```json
{
  "orderId": "400000123",
  "notificationId": "200000456",
  "status": "CRTD",
  "message": "Maintenance order created successfully",
  "createdAt": "2025-01-15T10:30:00Z"
}
```

#### Get Order Status

```bash
curl http://localhost:8080/api/v1/maintenance-orders/400000123
```

**Response (Simulator Mode):**
```json
{
  "orderId": "400000123",
  "status": "CRTD",
  "description": "Mock maintenance order",
  "equipmentId": "10000045",
  "plant": "1000",
  "notificationId": "200000123",
  "operations": [
    {
      "operationId": "0010",
      "text": "Mock operation",
      "status": "CNF",
      "actualWorkQuantity": 4.0,
      "workQuantityUnit": "H"
    }
  ]
}
```
### End-to-End Test

To run the complete integration test:
```bash
make test-simulator
```

This test creates a maintenance order, monitors status changes, and retrieves final equipment data.

### Simulator Behavior

The simulator generates realistic responses:

- **Notification IDs**: `200000XXX` format
- **Order IDs**: `400000XXX` format  
- **Status Progression**: CRTD → REL → TECO
- **Mock Operations**: Realistic operation data
- **Timestamps**: Current time for realistic testing

## Architecture

The service follows the following architecture pattern:

```
cmd/
├── main.go                 # Application entry point

internal/
├── config/                 # Configuration management
├── handlers/               # HTTP request handlers
├── services/               # Business logic
├── sap/                    # SAP API client
└── models/                 # Data models

api/
└── openapi.yaml           # OpenAPI specification
```