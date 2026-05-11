# SAP Adaptor

A Go service that bridges Digital Twin systems and SAP Plant Maintenance.

## Overview

1. Receives a maintenance order event from Digital Twin
2. Creates a SAP Maintenance Notification
3. Creates a SAP Maintenance Order referencing the notification
4. Polls order status until completion (TECO)
5. Sends a Maintenance Done event back to Digital Twin

## Running Locally

Build all binaries first:

```bash
make build-all
```

### Terminal 1 — SAP Simulator

The simulator mimics the SAP API. By default, orders wait for planner enrichment before progressing to completion.

```bash
./bin/sap-simulator
```

To run without requiring planner enrichment (orders auto-complete on a timer):

```bash
./bin/sap-simulator --planner-input=""
```

Runs on `http://localhost:8081`.

### Terminal 2 — SAP Adaptor

```bash
source env.example
./bin/sap-adaptor
```

Runs on `http://localhost:8080`, pointed at the simulator by default (`SAP_ADAPTOR_SAP_BASE_URL=http://localhost:8081`).

### Terminal 3 — Callback Listener (optional)

To receive and inspect the Digital Twin completion callbacks:

```bash
python3 scripts/listen-callback.py
```

Listens on `http://localhost:8082`.

## Simulator Behaviour

Orders progress through: `CRTD` → `REL` → `TECO`

**With planner enrichment required (default):**
- `CRTD` — immediately after creation; stays here until enriched
- `REL` — after `POST /planner/orders/{orderId}/enrich` is called on the simulator
- `TECO` — 30 seconds after enrichment

**Without planner enrichment (`--planner-input=""`):**
- `CRTD` — 0–10 seconds after creation
- `REL` — 10–30 seconds after creation
- `TECO` — 30+ seconds after creation

**Generated IDs:**
- Notification IDs: `200000XXX`
- Order IDs: `400000XXX`

## Running with Docker

1. Copy and configure the environment file:
```bash
cp env.docker .env
# Edit .env — set DIGITAL_TWIN_BASE_URL and DIGITAL_TWIN_API_KEY as needed
```

2. Start all services:
```bash
docker compose up -d
```

The simulator and adaptor start together. `SAP_SIMULATOR_PLANNER_INPUT=required` is the default in both `.env` and `docker-compose.yml`.

To disable planner enrichment, set `SAP_SIMULATOR_PLANNER_INPUT=` (empty) in `.env` before starting.

```bash
docker compose logs -f          # tail logs
docker compose down             # stop and remove containers
docker compose up --build -d    # rebuild and restart
```

## API Endpoints

### SAP Adaptor (`localhost:8080`)
- `POST /api/v1/maintenance-orders` — create a maintenance order event
- `GET  /api/v1/maintenance-orders/{id}` — get order status
- `POST /api/v1/maintenance-done` — handle maintenance completion
- `GET  /health`
- `GET  /swagger/index.html` — API docs

### SAP Simulator (`localhost:8081`)
- `POST /planner/orders/{orderId}/enrich` — enrich an order (when planner input required)
- `GET  /health`
