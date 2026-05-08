package sap

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"sap-adaptor/internal/models"
)

// StoredOrder holds the original order request data
type StoredOrder struct {
	Request    *models.SAPOrderRequest
	OrderID    string
	CreatedAt  time.Time
	Enrichment *models.PlannerEnrichmentRequest
	EnrichedAt *time.Time
}

var (
	ErrPlannerInputDisabled = errors.New("planner input mode is not enabled")
	ErrOrderNotFound        = errors.New("maintenance order not found")
	ErrOrderAlreadyEnriched = errors.New("maintenance order already enriched")
	ErrInvalidEnrichment    = errors.New("invalid planner enrichment request")
)

// MockGenerator provides functions to generate mock SAP responses
type MockGenerator struct {
	orderStore           map[string]*StoredOrder
	plannerInputRequired bool
	plannerTECODelay     time.Duration
	mu                   sync.RWMutex
}

// NewMockGenerator creates a new mock generator
func NewMockGenerator() *MockGenerator {
	return &MockGenerator{
		orderStore:       make(map[string]*StoredOrder),
		plannerTECODelay: 30 * time.Second,
	}
}

// SetPlannerInputRequired configures whether orders must wait for planner enrichment.
func (g *MockGenerator) SetPlannerInputRequired(required bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.plannerInputRequired = required
}

// PlannerInputRequired returns whether planner-input mode is active.
func (g *MockGenerator) PlannerInputRequired() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.plannerInputRequired
}

// SetPlannerTECODelay configures the delay from enrichment to TECO.
func (g *MockGenerator) SetPlannerTECODelay(delay time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.plannerTECODelay = delay
}

// CreateMockNotificationResponse creates a mock notification response for simulator mode
func (g *MockGenerator) CreateMockNotificationResponse(req *models.SAPNotificationRequest) *models.SAPNotificationResponse {
	// Generate a mock notification ID
	notificationID := fmt.Sprintf("200000%03d", time.Now().Unix()%1000)

	return &models.SAPNotificationResponse{
		D: struct {
			Notification string `json:"Notification"`
			Description  string `json:"Description"`
			Plant        string `json:"Plant"`
		}{
			Notification: notificationID,
			Description:  req.Description,
			Plant:        req.Plant,
		},
	}
}

// CreateMockOrderResponse creates a mock order response for simulator mode
func (g *MockGenerator) CreateMockOrderResponse(req *models.SAPOrderRequest) *models.SAPOrderResponse {
	// Generate a mock order ID
	orderID := fmt.Sprintf("400000%03d", time.Now().Unix()%1000)

	// Create mock operations
	var operations []models.SAPOrderOperationResponse
	for i, op := range req.ToMaintenanceOrderOperation {
		operationID := fmt.Sprintf("%04d", (i+1)*10)
		operations = append(operations, models.SAPOrderOperationResponse{
			MaintenanceOrder:          orderID,
			MaintenanceOrderOperation: operationID,
			OperationText:             op.OperationText,
			WorkCenter:                op.WorkCenter,
			OperationControlKey:       op.OperationControlKey,
			OperationStandardDuration: op.OperationStandardDuration,
			OperationDurationUnit:     op.OperationDurationUnit,
			Metadata: struct {
				ID   string `json:"id"`
				URI  string `json:"uri"`
				Type string `json:"type"`
			}{
				ID:   fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='%s')", orderID, operationID),
				URI:  fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='%s')", orderID, operationID),
				Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderOperationType",
			},
		})
	}

	// Create response with just operations (no components/object list on creation)
	resp := &models.SAPOrderResponse{}
	resp.D.MaintenanceOrder = orderID
	resp.D.MaintenanceOrderType = req.MaintenanceOrderType
	resp.D.Description = req.Description
	resp.D.Equipment = req.Equipment
	resp.D.Plant = req.Plant
	resp.D.OrderStatus = "CRTD" // Created status
	resp.D.MaintOrdBasicStartDateTime = req.MaintOrdBasicStartDateTime
	resp.D.MaintOrdBasicEndDateTime = req.MaintOrdBasicEndDateTime
	resp.D.MaintenanceNotification = req.MaintenanceNotification
	resp.D.Metadata.ID = fmt.Sprintf(".../A_MaintenanceOrder('%s')", orderID)
	resp.D.Metadata.URI = fmt.Sprintf(".../A_MaintenanceOrder('%s')", orderID)
	resp.D.Metadata.Type = "API_MAINTENANCE_ORDER.A_MaintenanceOrderType"
	resp.D.ToMaintenanceOrderOperation.Results = operations

	// Store the order for later retrieval
	g.mu.Lock()
	g.orderStore[orderID] = &StoredOrder{
		Request:   req,
		OrderID:   orderID,
		CreatedAt: time.Now(),
	}
	g.mu.Unlock()

	return resp
}

// EnrichOrder stores planner-provided operations, components, and optional actuals for an order.
func (g *MockGenerator) EnrichOrder(orderID string, req *models.PlannerEnrichmentRequest) (*models.PlannerEnrichmentResponse, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.plannerInputRequired {
		return nil, ErrPlannerInputDisabled
	}

	if err := validatePlannerEnrichment(req); err != nil {
		return nil, err
	}

	storedOrder := g.orderStore[orderID]
	if storedOrder == nil {
		return nil, ErrOrderNotFound
	}

	if storedOrder.Enrichment != nil {
		return nil, ErrOrderAlreadyEnriched
	}

	enrichedAt := time.Now()
	storedOrder.Enrichment = normalizePlannerEnrichment(req)
	storedOrder.EnrichedAt = &enrichedAt

	resp := &models.PlannerEnrichmentResponse{
		MaintenanceOrder: orderID,
		Status:           "REL",
		Message:          "Maintenance order enriched successfully",
		EnrichedAt:       enrichedAt,
	}

	for i, op := range storedOrder.Enrichment.Operations {
		operationID := op.Operation
		if operationID == "" {
			operationID = fmt.Sprintf("%04d", (i+1)*10)
		}
		resp.Operations = append(resp.Operations, models.PlannerEnrichmentOperationState{
			Operation: operationID,
			Status:    "REL",
		})
	}

	return resp, nil
}

// CreateMockOrderStatusResponse creates a mock order status response for simulator mode
// The status progresses over time:
// - 0-10 seconds: CRTD (Created)
// - 10-30 seconds: REL (Released)
// - 30+ seconds: TECO (Technically Completed) - Includes component and object list data
func (g *MockGenerator) CreateMockOrderStatusResponse(orderID string) *models.SAPOrderResponse {
	// Try to retrieve stored order data
	g.mu.RLock()
	storedOrder := g.orderStore[orderID]
	plannerInputRequired := g.plannerInputRequired
	plannerTECODelay := g.plannerTECODelay
	g.mu.RUnlock()

	// Determine status based on time elapsed since creation
	status := "CRTD" // Default to created
	isTECO := false

	if storedOrder != nil {
		if plannerInputRequired {
			if storedOrder.EnrichedAt != nil {
				status = "REL"
				if time.Since(*storedOrder.EnrichedAt) >= plannerTECODelay {
					status = "TECO"
					isTECO = true
				}
			}
		} else {
			elapsed := time.Since(storedOrder.CreatedAt)
			if elapsed >= 30*time.Second {
				status = "TECO"
				isTECO = true
			} else if elapsed >= 10*time.Second {
				status = "REL"
			}
		}
	}

	// Use stored order data if available, otherwise use defaults
	var operations []models.SAPOrderOperationResponse
	description := "Equipment replacement maintenance"
	equipment := "10000045"
	plant := "1000"
	maintenanceOrderType := "PM01"
	notification := "200000123"
	startTime := time.Now().Add(-8 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Format(time.RFC3339)

	if storedOrder != nil {
		// Use data from stored order
		req := storedOrder.Request
		description = req.Description
		equipment = req.Equipment
		plant = req.Plant
		maintenanceOrderType = req.MaintenanceOrderType
		notification = req.MaintenanceNotification
		startTime = req.MaintOrdBasicStartDateTime
		endTime = req.MaintOrdBasicEndDateTime

		if storedOrder.Enrichment != nil {
			if storedOrder.Enrichment.PlannedStartDateTime != "" {
				startTime = storedOrder.Enrichment.PlannedStartDateTime
			}
			if storedOrder.Enrichment.PlannedEndDateTime != "" {
				endTime = storedOrder.Enrichment.PlannedEndDateTime
			}
			operations = g.operationsFromPlannerEnrichment(orderID, storedOrder.Enrichment, isTECO)
		} else {
			// Generate operations from stored request
			for i, op := range req.ToMaintenanceOrderOperation {
				operationID := fmt.Sprintf("%04d", (i+1)*10)
				operations = append(operations, models.SAPOrderOperationResponse{
					MaintenanceOrder:          orderID,
					MaintenanceOrderOperation: operationID,
					OperationText:             op.OperationText,
					WorkCenter:                op.WorkCenter,
					OperationControlKey:       op.OperationControlKey,
					OperationStandardDuration: op.OperationStandardDuration,
					OperationDurationUnit:     op.OperationDurationUnit,
					OperationStatus:           "CNF",
					ActualWorkQuantity:        op.OperationStandardDuration, // Use planned as actual for simplicity
					WorkQuantityUnit:          op.OperationDurationUnit,
					Metadata: struct {
						ID   string `json:"id"`
						URI  string `json:"uri"`
						Type string `json:"type"`
					}{
						ID:   fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='%s')", orderID, operationID),
						URI:  fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='%s')", orderID, operationID),
						Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderOperationType",
					},
				})
			}
		}
	}

	// If no stored operations or no stored order, create default operations
	if len(operations) == 0 {
		operations = []models.SAPOrderOperationResponse{
			{
				MaintenanceOrder:          orderID,
				MaintenanceOrderOperation: "0010",
				OperationText:             "Disassemble and inspect equipment",
				WorkCenter:                "MECH-WC01",
				OperationControlKey:       "PM01",
				OperationStandardDuration: "2",
				OperationDurationUnit:     "H",
				OperationStatus:           "CNF",
				ActualWorkQuantity:        "2.0",
				WorkQuantityUnit:          "H",
				Metadata: struct {
					ID   string `json:"id"`
					URI  string `json:"uri"`
					Type string `json:"type"`
				}{
					ID:   fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='0010')", orderID),
					URI:  fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='0010')", orderID),
					Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderOperationType",
				},
			},
			{
				MaintenanceOrder:          orderID,
				MaintenanceOrderOperation: "0020",
				OperationText:             "Replace equipment and test",
				WorkCenter:                "MECH-WC01",
				OperationControlKey:       "PM01",
				OperationStandardDuration: "4",
				OperationDurationUnit:     "H",
				OperationStatus:           "CNF",
				ActualWorkQuantity:        "4.5",
				WorkQuantityUnit:          "H",
				Metadata: struct {
					ID   string `json:"id"`
					URI  string `json:"uri"`
					Type string `json:"type"`
				}{
					ID:   fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='0020')", orderID),
					URI:  fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='0020')", orderID),
					Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderOperationType",
				},
			},
		}
	}

	resp := &models.SAPOrderResponse{}
	resp.D.MaintenanceOrder = orderID
	resp.D.MaintenanceOrderType = maintenanceOrderType
	resp.D.Description = description
	resp.D.Equipment = equipment
	resp.D.Plant = plant
	resp.D.OrderStatus = status
	resp.D.MaintOrdBasicStartDateTime = startTime
	resp.D.MaintOrdBasicEndDateTime = endTime
	resp.D.MaintenanceNotification = notification
	resp.D.Metadata.ID = fmt.Sprintf(".../A_MaintenanceOrder('%s')", orderID)
	resp.D.Metadata.URI = fmt.Sprintf(".../A_MaintenanceOrder('%s')", orderID)
	resp.D.Metadata.Type = "API_MAINTENANCE_ORDER.A_MaintenanceOrderType"
	resp.D.ToMaintenanceOrderOperation.Results = operations

	// Add component and object list data only for TECO/CLSD orders
	if isTECO && storedOrder != nil && storedOrder.Enrichment != nil {
		resp.D.ToMaintOrderObjectListItem.Results = g.generateObjectList(orderID, equipment, storedOrder)
	} else if isTECO {
		// Generate components and attach to operations (real SAP structure)
		components := g.generateComponents(orderID, equipment, plant)
		// Attach components to the relevant operation (typically operation 0020)
		for i := range resp.D.ToMaintenanceOrderOperation.Results {
			if resp.D.ToMaintenanceOrderOperation.Results[i].MaintenanceOrderOperation == "0020" {
				resp.D.ToMaintenanceOrderOperation.Results[i].ToMaintOrderOpComponent2.Results = components
				break
			}
		}

		// Generate object list at order level
		objectList := g.generateObjectList(orderID, equipment, storedOrder)
		resp.D.ToMaintOrderObjectListItem.Results = objectList
	}

	return resp
}

func validatePlannerEnrichment(req *models.PlannerEnrichmentRequest) error {
	if req == nil || len(req.Operations) == 0 {
		return ErrInvalidEnrichment
	}

	for _, op := range req.Operations {
		if strings.TrimSpace(op.Description) == "" ||
			strings.TrimSpace(op.WorkCenter) == "" ||
			strings.TrimSpace(op.Plant) == "" ||
			strings.TrimSpace(op.PlannedWorkQuantity) == "" ||
			strings.TrimSpace(op.WorkQuantityUnit) == "" {
			return ErrInvalidEnrichment
		}

		for _, component := range op.Components {
			if strings.TrimSpace(component.Material) == "" ||
				strings.TrimSpace(component.RequiredQuantity) == "" ||
				strings.TrimSpace(component.Unit) == "" ||
				strings.TrimSpace(component.Plant) == "" {
				return ErrInvalidEnrichment
			}
		}
	}

	return nil
}

func normalizePlannerEnrichment(req *models.PlannerEnrichmentRequest) *models.PlannerEnrichmentRequest {
	normalized := *req
	normalized.Operations = make([]models.EnrichedOperation, len(req.Operations))

	for i, op := range req.Operations {
		normalizedOp := op
		if normalizedOp.Operation == "" {
			normalizedOp.Operation = fmt.Sprintf("%04d", (i+1)*10)
		}
		if normalizedOp.ControlKey == "" {
			normalizedOp.ControlKey = "PM01"
		}
		if normalizedOp.PlannedDuration == "" {
			normalizedOp.PlannedDuration = normalizedOp.PlannedWorkQuantity
		}
		if normalizedOp.DurationUnit == "" {
			normalizedOp.DurationUnit = normalizedOp.WorkQuantityUnit
		}

		normalizedOp.Components = make([]models.EnrichedComponent, len(op.Components))
		for j, component := range op.Components {
			normalizedComponent := component
			if normalizedComponent.Component == "" {
				normalizedComponent.Component = fmt.Sprintf("%04d", j+1)
			}
			if normalizedComponent.ItemCategory == "" {
				normalizedComponent.ItemCategory = "L"
			}
			if normalizedComponent.GoodsMovementType == "" {
				normalizedComponent.GoodsMovementType = "261"
			}
			normalizedOp.Components[j] = normalizedComponent
		}

		normalized.Operations[i] = normalizedOp
	}

	return &normalized
}

func (g *MockGenerator) operationsFromPlannerEnrichment(orderID string, enrichment *models.PlannerEnrichmentRequest, isTECO bool) []models.SAPOrderOperationResponse {
	operations := make([]models.SAPOrderOperationResponse, 0, len(enrichment.Operations))

	for _, op := range enrichment.Operations {
		operation := models.SAPOrderOperationResponse{
			MaintenanceOrder:          orderID,
			MaintenanceOrderOperation: op.Operation,
			OperationText:             op.Description,
			WorkCenter:                op.WorkCenter,
			OperationControlKey:       op.ControlKey,
			OperationStandardDuration: op.PlannedDuration,
			OperationDurationUnit:     op.DurationUnit,
			OperationStatus:           "REL",
			WorkQuantityUnit:          op.WorkQuantityUnit,
			Metadata: struct {
				ID   string `json:"id"`
				URI  string `json:"uri"`
				Type string `json:"type"`
			}{
				ID:   fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='%s')", orderID, op.Operation),
				URI:  fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='%s')", orderID, op.Operation),
				Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderOperationType",
			},
		}

		if isTECO {
			operation.OperationStatus = "CNF"
			operation.ActualWorkQuantity = op.ActualWorkQuantity
			if operation.ActualWorkQuantity == "" {
				operation.ActualWorkQuantity = op.PlannedWorkQuantity
			}
			operation.OpActualExecutionStartDateTime = op.ActualStartDateTime
			operation.OpActualExecutionEndDateTime = op.ActualEndDateTime
			operation.ToMaintOrderOpComponent2.Results = g.componentsFromPlannerOperation(orderID, op)
		}

		operations = append(operations, operation)
	}

	return operations
}

func (g *MockGenerator) componentsFromPlannerOperation(orderID string, op models.EnrichedOperation) []models.SAPOrderComponentResponse {
	components := make([]models.SAPOrderComponentResponse, 0, len(op.Components))
	reservationBase := fmt.Sprintf("%010d", time.Now().Unix()%10000000)

	for i, component := range op.Components {
		quantity := component.UsedQuantity
		if quantity == "" {
			quantity = component.RequiredQuantity
		}

		finalIssue := false
		if component.FinalIssue != nil {
			finalIssue = *component.FinalIssue
		}

		reservationItem := fmt.Sprintf("%04d", i+1)
		components = append(components, models.SAPOrderComponentResponse{
			MaintenanceOrder:               orderID,
			MaintenanceOrderOperation:      op.Operation,
			MaintenanceOrderSubOperation:   "0000",
			MaintenanceOrderComponent:      component.Component,
			Product:                        component.Material,
			MaintOrdOperationComponentText: component.Description,
			MaintOrdOpCompRequiredQuantity: quantity,
			BaseUnit:                       component.Unit,
			MaintComponentItemCategory:     component.ItemCategory,
			GoodsMovementType:              component.GoodsMovementType,
			Plant:                          component.Plant,
			StorageLocation:                component.StorageLocation,
			Reservation:                    reservationBase,
			ReservationItem:                reservationItem,
			ReservationIsFinallyIssued:     finalIssue,
			Metadata: struct {
				ID   string `json:"id"`
				URI  string `json:"uri"`
				Type string `json:"type"`
			}{
				ID:   fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='%s')", orderID, component.Component),
				URI:  fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='%s')", orderID, component.Component),
				Type: "API_MAINTENANCE_ORDER.MaintOrderOpComponent_Type",
			},
		})
	}

	return components
}

// generateComponents creates component list based on equipment type
func (g *MockGenerator) generateComponents(orderID, equipment, plant string) []models.SAPOrderComponentResponse {
	var components []models.SAPOrderComponentResponse
	reservationBase := fmt.Sprintf("%010d", time.Now().Unix()%10000000)

	// Determine component type based on equipment
	equipmentUpper := strings.ToUpper(equipment)

	if strings.Contains(equipmentUpper, "PUMP") {
		// Pump-related components
		components = []models.SAPOrderComponentResponse{
			{
				MaintenanceOrder:               orderID,
				MaintenanceOrderOperation:      "0020",
				MaintenanceOrderSubOperation:   "0000",
				MaintenanceOrderComponent:      "0001",
				Product:                        "PUMP-SEAL-X200",
				MaintOrdOperationComponentText: "High-performance pump seal",
				MaintOrdOpCompRequiredQuantity: "1.000",
				BaseUnit:                       "EA",
				MaintComponentItemCategory:     "L",
				GoodsMovementType:              "261",
				Plant:                          plant,
				StorageLocation:                "0001",
				Reservation:                    reservationBase,
				ReservationItem:                "0001",
				ReservationIsFinallyIssued:     true,
				Metadata: struct {
					ID   string `json:"id"`
					URI  string `json:"uri"`
					Type string `json:"type"`
				}{
					ID:   fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0001')", orderID),
					URI:  fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0001')", orderID),
					Type: "API_MAINTENANCE_ORDER.MaintOrderOpComponent_Type",
				},
			},
			{
				MaintenanceOrder:               orderID,
				MaintenanceOrderOperation:      "0020",
				MaintenanceOrderSubOperation:   "0000",
				MaintenanceOrderComponent:      "0002",
				Product:                        "BEARING-KIT-450",
				MaintOrdOperationComponentText: "Bearing kit for pump",
				MaintOrdOpCompRequiredQuantity: "1.000",
				BaseUnit:                       "EA",
				MaintComponentItemCategory:     "L",
				GoodsMovementType:              "261",
				Plant:                          plant,
				StorageLocation:                "0001",
				Reservation:                    reservationBase,
				ReservationItem:                "0002",
				ReservationIsFinallyIssued:     true,
				Metadata: struct {
					ID   string `json:"id"`
					URI  string `json:"uri"`
					Type string `json:"type"`
				}{
					ID:   fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0002')", orderID),
					URI:  fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0002')", orderID),
					Type: "API_MAINTENANCE_ORDER.MaintOrderOpComponent_Type",
				},
			},
		}
	} else if strings.Contains(equipmentUpper, "VALVE") {
		// Valve-related components
		components = []models.SAPOrderComponentResponse{
			{
				MaintenanceOrder:               orderID,
				MaintenanceOrderOperation:      "0020",
				MaintenanceOrderSubOperation:   "0000",
				MaintenanceOrderComponent:      "0001",
				Product:                        "VALVE-GASKET-V50",
				MaintOrdOperationComponentText: "Valve gasket set",
				MaintOrdOpCompRequiredQuantity: "2.000",
				BaseUnit:                       "EA",
				MaintComponentItemCategory:     "L",
				GoodsMovementType:              "261",
				Plant:                          plant,
				StorageLocation:                "0001",
				Reservation:                    reservationBase,
				ReservationItem:                "0001",
				ReservationIsFinallyIssued:     true,
				Metadata: struct {
					ID   string `json:"id"`
					URI  string `json:"uri"`
					Type string `json:"type"`
				}{
					ID:   fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0001')", orderID),
					URI:  fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0001')", orderID),
					Type: "API_MAINTENANCE_ORDER.MaintOrderOpComponent_Type",
				},
			},
			{
				MaintenanceOrder:               orderID,
				MaintenanceOrderOperation:      "0020",
				MaintenanceOrderSubOperation:   "0000",
				MaintenanceOrderComponent:      "0002",
				Product:                        "VALVE-STEM-KIT",
				MaintOrdOperationComponentText: "Valve stem repair kit",
				MaintOrdOpCompRequiredQuantity: "1.000",
				BaseUnit:                       "EA",
				MaintComponentItemCategory:     "L",
				GoodsMovementType:              "261",
				Plant:                          plant,
				StorageLocation:                "0001",
				Reservation:                    reservationBase,
				ReservationItem:                "0002",
				ReservationIsFinallyIssued:     true,
				Metadata: struct {
					ID   string `json:"id"`
					URI  string `json:"uri"`
					Type string `json:"type"`
				}{
					ID:   fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0002')", orderID),
					URI:  fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0002')", orderID),
					Type: "API_MAINTENANCE_ORDER.MaintOrderOpComponent_Type",
				},
			},
		}
	} else {
		// Generic components for other equipment types
		components = []models.SAPOrderComponentResponse{
			{
				MaintenanceOrder:               orderID,
				MaintenanceOrderOperation:      "0020",
				MaintenanceOrderSubOperation:   "0000",
				MaintenanceOrderComponent:      "0001",
				Product:                        "GENERIC-PART-001",
				MaintOrdOperationComponentText: "Replacement part",
				MaintOrdOpCompRequiredQuantity: "1.000",
				BaseUnit:                       "EA",
				MaintComponentItemCategory:     "L",
				GoodsMovementType:              "261",
				Plant:                          plant,
				StorageLocation:                "0001",
				Reservation:                    reservationBase,
				ReservationItem:                "0001",
				ReservationIsFinallyIssued:     true,
				Metadata: struct {
					ID   string `json:"id"`
					URI  string `json:"uri"`
					Type string `json:"type"`
				}{
					ID:   fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0001')", orderID),
					URI:  fmt.Sprintf(".../A_MaintenanceOrderComponent(MaintenanceOrder='%s',Component='0001')", orderID),
					Type: "API_MAINTENANCE_ORDER.MaintOrderOpComponent_Type",
				},
			},
		}
	}

	return components
}

// generateObjectList creates object list based on equipment
func (g *MockGenerator) generateObjectList(orderID, equipment string, storedOrder *StoredOrder) []models.SAPObjectListItemResponse {
	// Generate unique identifiers for this specific order
	randomSuffix := time.Now().Unix() % 100000
	newSerialNumber := fmt.Sprintf("SN-2026-%05d", randomSuffix)
	newEquipmentNumber := fmt.Sprintf("%s-NEW-%05d", equipment, randomSuffix)

	// Determine material and assembly based on equipment type
	equipmentUpper := strings.ToUpper(equipment)
	material := "GENERIC-MODEL"
	assembly := "ASSEMBLY-01"
	functionalLocation := "FL100-200-300"

	if strings.Contains(equipmentUpper, "PUMP") {
		material = "PUMP-MODEL-X200"
		assembly = "PUMP-ASSEMBLY-01"
	} else if strings.Contains(equipmentUpper, "VALVE") {
		material = "VALVE-MODEL-V50"
		assembly = "VALVE-ASSEMBLY-01"
	}

	// Use functional location from stored order if available
	if storedOrder != nil && storedOrder.Request.FunctionalLocation != "" {
		functionalLocation = storedOrder.Request.FunctionalLocation
	}

	return []models.SAPObjectListItemResponse{
		{
			MaintenanceOrder:            orderID,
			MaintenanceObjectListItem:   1,
			Equipment:                   newEquipmentNumber,
			Material:                    material,
			SerialNumber:                newSerialNumber,
			Assembly:                    assembly,
			FunctionalLocation:          functionalLocation,
			MaintObjectListItemSequence: "0001",
			Metadata: struct {
				ID   string `json:"id"`
				URI  string `json:"uri"`
				Type string `json:"type"`
			}{
				ID:   fmt.Sprintf(".../A_MaintOrderObjectListItem(MaintenanceOrder='%s',Item=1)", orderID),
				URI:  fmt.Sprintf(".../A_MaintOrderObjectListItem(MaintenanceOrder='%s',Item=1)", orderID),
				Type: "API_MAINTENANCE_ORDER.MaintOrderObjectListItemType",
			},
		},
	}
}
