# SAP Adaptor - Quick Start Guide

## One-Command Setup

```bash
# Clone the repository
git clone <your-repo-url>
cd "SAP Adaptor"

# Run the setup script
./setup.sh
```

The SAP Adaptor will be running on `http://localhost:8080`

## Configuration

Edit `.env` file to configure:

### Server Settings
- `SAP_ADAPTOR_SERVER_PORT=8080` - Change the port if needed
- `SAP_ADAPTOR_SERVER_HOST=0.0.0.0` - Server host (usually keep as is)

### SAP Integration
- **Simulator Mode** (default): `SAP_ADAPTOR_SAP_SIMULATOR_MODE=true`
- **Real SAP**: Set `SAP_ADAPTOR_SAP_SIMULATOR_MODE=false` and configure credentials

### Digital Twin Integration
- `SAP_ADAPTOR_DIGITAL_TWIN_BASE_URL` - Your Digital Twin API URL
- `SAP_ADAPTOR_DIGITAL_TWIN_API_KEY` - API key for authentication

## API Endpoints

- **Health Check**: `GET /health`
- **Create Order**: `POST /api/v1/maintenance-orders`
- **Get Order**: `GET /api/v1/maintenance-orders/{id}`
- **Maintenance Done**: `POST /api/v1/maintenance-done`

## API Documentation

- **Swagger UI**: http://localhost:8080/swagger/index.html
- **OpenAPI Spec**: http://localhost:8080/swagger/doc.json

## Development Commands

```bash
# View logs
docker-compose logs -f

# Restart service
docker-compose restart

# Stop service
docker-compose down

# Rebuild and start
docker-compose up --build -d
```

## Testing

```bash
# Run end-to-end test
make test-simulator
```

## Project Structure

```
├── api/                    # OpenAPI specifications
├── cmd/                    # Executable applications
│   ├── server/            # Main API server
│   ├── simulator/         # SAP simulator
│   └── test/              # End-to-end test
├── internal/              # Internal packages
│   ├── handlers/          # HTTP handlers
│   ├── services/          # Business logic
│   ├── models/            # Data models
│   └── sap/              # SAP client
├── docker-compose.yml     # Container orchestration
├── Dockerfile            # Container build
└── setup.sh             # Quick setup script
```