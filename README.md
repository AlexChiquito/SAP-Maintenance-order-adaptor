# SAP Adaptor

A Go-based service that acts as a bridge between Digital Twin Event Generator systems and SAP Plant Maintenance.

## Overview

The SAP Adaptor follows the integration workflow specified in the architecture:

1. **Receives Maintenance Order Event** from Digital Twin
2. **Creates SAP Maintenance Notification** 
3. **Creates SAP Maintenance Order** with notification reference
4. **Monitors order status** until completion
5. **Sends Maintenance Done Event** back to Digital Twin

## Features

- RESTful API with OpenAPI 3.0 specification
- SAP Plant Maintenance integration (with simulator mode for testing)
- **Simulator Mode** for testing without real SAP access

## API Endpoints

### Maintenance Orders
- `POST /api/v1/maintenance-orders` - Create maintenance order event
- `GET /api/v1/maintenance-orders/{id}` - Get maintenance order status

### Maintenance Events  
- `POST /api/v1/maintenance-done` - Handle maintenance completion event

### System
- `GET /health` - Health check
- `GET /metrics` - Service metrics

## Quick Start

### Prerequisites
- Go 1.21 or later
- Access to SAP Plant Maintenance system (optional - simulator mode available)
- Digital Twin system (optional for testing)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd sap-adaptor
```

2. Install dependencies:
```bash
go mod tidy
```
3. Use the setup script:
```bash
./setup.sh
````
### Manual installation
1. Clone the repository:
```bash
git clone <repository-url>
cd sap-adaptor
```

2. Install dependencies:
```bash
go mod tidy
```

3. Configure environment variables:
```bash
cp env.example .env
# Edit .env with your configuration (simulator mode is enabled by default)
```

4. Run the service:
```bash
go run ./cmd/server
```

The service will start on `http://localhost:8080` in **simulator mode** by default.

### Docker

```bash
# Build the image
docker build -t sap-adaptor .

# Run the container
docker run -p 8080:8080 --env-file .env sap-adaptor
```

## Configuration

The service can be configured via:

1. **Environment variables** (recommended for production)
2. **YAML configuration file** (`config.yaml`)
3. **Command line flags** (future enhancement)

### Simulator Mode (Default)

For testing and demonstration purposes, the service runs in **simulator mode** by default:

- **Mock responses** for all SAP API calls
- **Realistic data** that follows SAP response formats

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

## Testing with Simulator Mode

The simulator mode can be used for testing the API structure and workflow without needing access to a real SAP system.

### Quick Test Script

Run the included test script to see the full workflow:

```bash
./scripts/test-api.sh
```

This script will:
1. Test the health endpoint
2. Create a maintenance order (returns mock SAP order ID)
3. Query the order status
4. Send a maintenance done event

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
### Demo
To test the end-to-end simulator demo, use:
```bash
make test-simulator
```

### Simulator Behavior

The simulator generates realistic responses:

- **Notification IDs**: `200000XXX` format
- **Order IDs**: `400000XXX` format  
- **Status Progression**: CRTD → REL → TECO → CLSD (based on order ID)
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