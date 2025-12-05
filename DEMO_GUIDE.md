# SAP Adaptor Data Flow Demo

This demo showcases the complete data flow between the Digital Twin Event Generator, SAP Adaptor, and SAP Simulator systems, highlighting the new architecture with real HTTP communication.

## Architecture Overview

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│  Digital Twin   │────────▶│  SAP Adaptor    │────────▶│  SAP Simulator  │
│ Event Generator │  (1)    │   (Port 8080)   │  (2)    │   (Port 8081)   │
└─────────────────┘         └─────────────────┘         └─────────────────┘
         ▲                           │
         │                           │
         └───────────────────────────┘
                    (3)
```

### Data Flow Steps:

1. **Digital Twin → SAP Adaptor**: Sends Maintenance Order Event (JSON)
2. **SAP Adaptor → SAP Simulator**: 
   - POST Notification (HTTP)
   - POST Order (HTTP)
   - GET Order Status (HTTP polling)
3. **SAP Adaptor → Digital Twin**: Sends Maintenance Completed Event (JSON)

## Running the Demo

### Option 1: Using Docker Compose (Recommended)

This is the easiest way to see the full system in action:

```bash
# Start both services
docker compose up -d

# Wait a few seconds for services to start, then run the demo
./bin/demo-flow

# View logs
docker compose logs -f

# Stop services
docker compose down
```

### Option 2: Manual Setup

Run each component in a separate terminal:

**Terminal 1 - Start SAP Simulator:**
```bash
./bin/simulator
# Or: go run ./cmd/simulator/main.go
```

**Terminal 2 - Run the Demo:**
```bash
./bin/demo-flow
# Or: go run ./cmd/demo-flow/main.go
```

**Terminal 3 (Optional) - Start SAP Adaptor API:**
```bash
./bin/adaptor
# Or: go run ./cmd/server/main.go
```

## What the Demo Shows

The demo will display:

### 1. Incoming Event from Digital Twin
- Equipment details
- Maintenance operations
- Scheduling information

### 2. SAP Adaptor Processing
- **Step 2.1**: Creates notification via HTTP POST
- **Step 2.2**: Creates maintenance order via HTTP POST
- **Step 2.3**: Verifies order creation via HTTP GET

### 3. Status Monitoring
- Polls SAP every 30 seconds (simulated faster in demo)
- Shows HTTP GET requests
- Displays current order status

### 4. Completion Event
- Shows the event sent back to Digital Twin
- Includes order details and completion timestamp

## Example Output

```
╔═══════════════════════════════════════════════════════════════╗
║         SAP Adaptor - Data Flow Demonstration                ║
║    Showing HTTP communication between systems                ║
╚═══════════════════════════════════════════════════════════════╝

┌─────────────────────────────────────────────────────────────┐
│ STEP 1: Digital Twin Event Generator sends Maintenance Order Event
└─────────────────────────────────────────────────────────────┘

   📋 Incoming Event:
      Equipment ID: 10000045
      Plant: 1000
      Description: Pump maintenance - bearing replacement
      Priority: 3
      Operations: 2
        1. Replace bearings (4.0 H)
        2. Lubricate pump (1.0 H)

...
```

## Key Features Demonstrated

### New Architecture Benefits:
- ✅ **Separate Services**: Simulator runs independently on port 8081
- ✅ **Real HTTP**: Actual network calls between services
- ✅ **Container Ready**: Can be deployed in separate containers
- ✅ **Production Ready**: Easy to swap simulator with real SAP

### HTTP Endpoints Used:
1. `POST /API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification`
2. `POST /API_MAINTENANCE_ORDER/A_MaintenanceOrder`
3. `GET /API_MAINTENANCE_ORDER/A_MaintenanceOrder('{id}')`

## Building the Demo

```bash
# Build all components
make build

# Or build individually
go build -o bin/simulator ./cmd/simulator
go build -o bin/adaptor ./cmd/server
go build -o bin/demo-flow ./cmd/demo-flow
```

## Troubleshooting

**Error: "connection refused" or "no such host"**
- Make sure the simulator is running first
- Check that it's running on port 8081: `curl http://localhost:8081/health`

**Error: "cannot find package"**
- Run: `go mod tidy`
- Then rebuild: `go build -o bin/demo-flow ./cmd/demo-flow`

## Other Demos

- **demo-polling.go**: Original demo showing status polling mechanism
- **demo-flow** (this demo): Shows complete HTTP data flow between systems

## Next Steps

After running the demo:
1. Examine the logs to see the HTTP requests and responses
2. Try modifying the event data in `cmd/demo-flow/main.go`
3. Deploy using Docker Compose for a production-like environment
4. Replace the simulator URL with a real SAP endpoint when ready
