# Planning Enrichment Flow

API draft: [`../api/openapi-planner-simulator.yaml`](../api/openapi-planner-simulator.yaml)

The draft only exposes the planner enrichment operation:
`POST /planner/orders/{maintenanceOrder}/enrich`. The payload is simplified for
planner use, while keeping SAP-oriented field names for operations, components,
planned work, and optional actual work.

## Simulator Mode
```bash
./sap-simulator --planner-input=required
```

## Process Flow

```mermaid
sequenceDiagram
    participant DT as Digital Twin
    participant Adaptor as SAP Adaptor
    participant Simulator as SAP Simulator
    participant Planner as Planner/System

    Note over DT,Planner: Initial Order Creation (Minimal Data)

    DT->>+Adaptor: POST /api/v1/maintenance-orders
    Note right of DT: equipmentId: 10000045<br/>description: Pump seal replacement

    Adaptor->>+Simulator: POST /API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification
    Simulator-->>-Adaptor: 201 Created
    Note left of Simulator: Notification: 200000586

    Adaptor->>+Simulator: POST /API_MAINTENANCE_ORDER/A_MaintenanceOrder
    Note right of Adaptor: Basic order only<br/>No operations yet<br/>No materials yet
    
    Simulator-->>-Adaptor: 201 Created
    Note left of Simulator: Order: 400000586<br/>Status: CRTD

    Adaptor-->>-DT: 200 OK
    Note left of Adaptor: Order created:<br/>400000586

    Note over DT,Planner: Planning Phase - Order Stays in CRTD

    Note over Simulator: ORDER WAITS IN CRTD<br/>Needs planning data...<br/>(No automatic progression)

    Note over Planner: Planner gathers data from:<br/>- Digital Twin (asset history)<br/>- Inventory (spare parts)<br/>- 3rd party systems

    Planner->>+Simulator: POST /planner/orders/400000586/enrich
    Note right of Planner: Planning data:<br/>{<br/>  "operations": [{<br/>    "operationId": "0010",<br/>    "description": "Disassemble pump",<br/>    "plannedWorkQuantity": "4.0",<br/>    "workQuantityUnit": "H",<br/>    "workCenter": "MECH-01"<br/>  }, {<br/>    "operationId": "0020",<br/>    "description": "Replace seal",<br/>    "plannedWorkQuantity": "3.0",<br/>    "workQuantityUnit": "H",<br/>    "workCenter": "MECH-01"<br/>  }],<br/>  "materials": [{<br/>    "material": "SEAL-X200",<br/>    "quantity": 1,<br/>    "plant": "1000"<br/>  }, {<br/>    "material": "BEARING-KIT",<br/>    "quantity": 1,<br/>    "plant": "1000"<br/>  }]<br/>}

    Simulator-->>-Planner: 200 OK
    Note left of Simulator: Status: CRTD → REL<br/>(Auto-released on enrichment)

    Note over DT,Planner: Work Execution Phase

    Note over Simulator: Time passes (20s)<br/>Status: REL → TECO<br/>(Simulating work execution)

    Note over DT,Planner: Status Check

    DT->>+Adaptor: GET /api/v1/maintenance-orders/400000586
    
    Adaptor->>+Simulator: GET /API_MAINTENANCE_ORDER/A_MaintenanceOrder('400000586')
    Note right of Adaptor: $expand: operations,<br/>components

    Simulator-->>-Adaptor: 200 OK
    Note left of Simulator: Status: TECO<br/>Op 0010: 4.0H<br/>Op 0020: 3.0H<br/>Materials: SEAL-X200, BEARING-KIT

    Adaptor-->>-DT: 200 OK

    Note over DT,Planner: Completion Notification

    Adaptor->>+DT: POST /api/v1/maintenance-completed
    Note right of Adaptor: Order 400000586 TECO

    DT-->>-Adaptor: 202 Accepted
```
