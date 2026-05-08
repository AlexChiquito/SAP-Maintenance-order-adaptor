package sap

import (
	"testing"
	"time"

	"sap-adaptor/internal/models"
)

func TestMockGenerator_CreateMockNotificationResponse(t *testing.T) {
	generator := NewMockGenerator()

	req := &models.SAPNotificationRequest{
		NotificationType:   "M1",
		Description:        "Test notification",
		Equipment:          "10000045",
		FunctionalLocation: "FL100-200-300",
		Plant:              "1000",
		Priority:           "3",
	}

	resp := generator.CreateMockNotificationResponse(req)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.D.Notification == "" {
		t.Error("Expected notification ID, got empty string")
	}

	if resp.D.Description != req.Description {
		t.Errorf("Expected description %s, got %s", req.Description, resp.D.Description)
	}

	if resp.D.Plant != req.Plant {
		t.Errorf("Expected plant %s, got %s", req.Plant, resp.D.Plant)
	}
}

func TestMockGenerator_CreateMockOrderResponse(t *testing.T) {
	generator := NewMockGenerator()

	req := &models.SAPOrderRequest{
		MaintenanceOrderType:       "PM01",
		Description:                "Test order",
		Equipment:                  "10000045",
		FunctionalLocation:         "FL100-200-300",
		Plant:                      "1000",
		MaintenanceNotification:    "200000123",
		Priority:                   "3",
		MaintOrdBasicStartDateTime: time.Now().Format(time.RFC3339),
		MaintOrdBasicEndDateTime:   time.Now().Add(8 * time.Hour).Format(time.RFC3339),
		ToMaintenanceOrderOperation: []models.SAPOrderOperation{
			{
				OperationText:             "Test operation",
				WorkCenter:                "TEST-WC01",
				OperationControlKey:       "PM01",
				OperationStandardDuration: "4",
				OperationDurationUnit:     "H",
			},
		},
	}

	resp := generator.CreateMockOrderResponse(req)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.D.MaintenanceOrder == "" {
		t.Error("Expected order ID, got empty string")
	}

	if resp.D.Description != req.Description {
		t.Errorf("Expected description %s, got %s", req.Description, resp.D.Description)
	}

	if resp.D.Equipment != req.Equipment {
		t.Errorf("Expected equipment %s, got %s", req.Equipment, resp.D.Equipment)
	}

	if resp.D.MaintenanceNotification != req.MaintenanceNotification {
		t.Errorf("Expected notification %s, got %s", req.MaintenanceNotification, resp.D.MaintenanceNotification)
	}

	if resp.D.OrderStatus != "CRTD" {
		t.Errorf("Expected status CRTD, got %s", resp.D.OrderStatus)
	}

	// Check operations
	if len(resp.D.ToMaintenanceOrderOperation.Results) != len(req.ToMaintenanceOrderOperation) {
		t.Errorf("Expected %d operations, got %d", len(req.ToMaintenanceOrderOperation), len(resp.D.ToMaintenanceOrderOperation.Results))
	}
}

func TestMockGenerator_CreateMockOrderStatusResponse(t *testing.T) {
	generator := NewMockGenerator()

	// Test 1: Unknown order should return CRTD
	t.Run("UnknownOrder", func(t *testing.T) {
		resp := generator.CreateMockOrderStatusResponse("UNKNOWN123")
		if resp == nil {
			t.Fatal("Expected response, got nil")
		}
		if resp.D.OrderStatus != "CRTD" {
			t.Errorf("Expected status CRTD for unknown order, got %s", resp.D.OrderStatus)
		}
	})

	// Test 2: Time-based status progression
	t.Run("TimeBasedProgression", func(t *testing.T) {
		// Create an order first
		req := &models.SAPOrderRequest{
			MaintenanceOrderType:    "PM01",
			Description:             "Test order",
			Equipment:               "10000045",
			FunctionalLocation:      "TEST-LOC",
			Plant:                   "1000",
			MaintenanceNotification: "200000123",
		}
		createResp := generator.CreateMockOrderResponse(req)
		orderID := createResp.D.MaintenanceOrder

		// Immediately after creation - should be CRTD
		statusResp := generator.CreateMockOrderStatusResponse(orderID)
		if statusResp.D.OrderStatus != "CRTD" {
			t.Errorf("Expected status CRTD immediately after creation, got %s", statusResp.D.OrderStatus)
		}

		generator.mu.Lock()
		generator.orderStore[orderID].CreatedAt = time.Now().Add(-11 * time.Second)
		generator.mu.Unlock()
		statusResp = generator.CreateMockOrderStatusResponse(orderID)
		if statusResp.D.OrderStatus != "REL" {
			t.Errorf("Expected status REL after 11 seconds, got %s", statusResp.D.OrderStatus)
		}

		generator.mu.Lock()
		generator.orderStore[orderID].CreatedAt = time.Now().Add(-31 * time.Second)
		generator.mu.Unlock()
		statusResp = generator.CreateMockOrderStatusResponse(orderID)
		if statusResp.D.OrderStatus != "TECO" {
			t.Errorf("Expected status TECO after 31 seconds, got %s", statusResp.D.OrderStatus)
		}
	})
}

func TestMockGenerator_PlannerInputStatusProgression(t *testing.T) {
	generator := NewMockGenerator()
	generator.SetPlannerInputRequired(true)

	createResp := generator.CreateMockOrderResponse(&models.SAPOrderRequest{
		MaintenanceOrderType:    "PM01",
		Description:             "Planner order",
		Equipment:               "PUMP-100",
		FunctionalLocation:      "TEST-LOC",
		Plant:                   "1000",
		MaintenanceNotification: "200000123",
	})
	orderID := createResp.D.MaintenanceOrder

	generator.mu.Lock()
	generator.orderStore[orderID].CreatedAt = time.Now().Add(-2 * time.Minute)
	generator.mu.Unlock()

	statusResp := generator.CreateMockOrderStatusResponse(orderID)
	if statusResp.D.OrderStatus != "CRTD" {
		t.Errorf("Expected planner-input order to remain CRTD before enrichment, got %s", statusResp.D.OrderStatus)
	}

	enrichResp, err := generator.EnrichOrder(orderID, samplePlannerEnrichmentRequest())
	if err != nil {
		t.Fatalf("Expected enrichment to succeed, got %v", err)
	}
	if enrichResp.Status != "REL" {
		t.Errorf("Expected enrichment response status REL, got %s", enrichResp.Status)
	}

	statusResp = generator.CreateMockOrderStatusResponse(orderID)
	if statusResp.D.OrderStatus != "REL" {
		t.Errorf("Expected order status REL after enrichment, got %s", statusResp.D.OrderStatus)
	}

	generator.mu.Lock()
	*generator.orderStore[orderID].EnrichedAt = time.Now().Add(-31 * time.Second)
	generator.mu.Unlock()

	statusResp = generator.CreateMockOrderStatusResponse(orderID)
	if statusResp.D.OrderStatus != "TECO" {
		t.Errorf("Expected order status TECO 30 seconds after enrichment, got %s", statusResp.D.OrderStatus)
	}
}

func TestMockGenerator_EnrichOrderErrors(t *testing.T) {
	t.Run("PlannerInputDisabled", func(t *testing.T) {
		generator := NewMockGenerator()
		_, err := generator.EnrichOrder("400000123", samplePlannerEnrichmentRequest())
		if err != ErrPlannerInputDisabled {
			t.Fatalf("Expected ErrPlannerInputDisabled, got %v", err)
		}
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		generator := NewMockGenerator()
		generator.SetPlannerInputRequired(true)
		_, err := generator.EnrichOrder("400000123", samplePlannerEnrichmentRequest())
		if err != ErrOrderNotFound {
			t.Fatalf("Expected ErrOrderNotFound, got %v", err)
		}
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		generator := NewMockGenerator()
		generator.SetPlannerInputRequired(true)
		createResp := generator.CreateMockOrderResponse(&models.SAPOrderRequest{Description: "Planner order", Plant: "1000"})
		_, err := generator.EnrichOrder(createResp.D.MaintenanceOrder, &models.PlannerEnrichmentRequest{})
		if err != ErrInvalidEnrichment {
			t.Fatalf("Expected ErrInvalidEnrichment, got %v", err)
		}
	})

	t.Run("AlreadyEnriched", func(t *testing.T) {
		generator := NewMockGenerator()
		generator.SetPlannerInputRequired(true)
		createResp := generator.CreateMockOrderResponse(&models.SAPOrderRequest{Description: "Planner order", Plant: "1000"})
		if _, err := generator.EnrichOrder(createResp.D.MaintenanceOrder, samplePlannerEnrichmentRequest()); err != nil {
			t.Fatalf("Expected first enrichment to succeed, got %v", err)
		}
		_, err := generator.EnrichOrder(createResp.D.MaintenanceOrder, samplePlannerEnrichmentRequest())
		if err != ErrOrderAlreadyEnriched {
			t.Fatalf("Expected ErrOrderAlreadyEnriched, got %v", err)
		}
	})
}

func TestMockGenerator_PlannerEnrichmentResponseMapping(t *testing.T) {
	generator := NewMockGenerator()
	generator.SetPlannerInputRequired(true)

	createResp := generator.CreateMockOrderResponse(&models.SAPOrderRequest{
		MaintenanceOrderType:    "PM01",
		Description:             "Planner order",
		Equipment:               "PUMP-100",
		FunctionalLocation:      "TEST-LOC",
		Plant:                   "1000",
		MaintenanceNotification: "200000123",
	})
	orderID := createResp.D.MaintenanceOrder

	if _, err := generator.EnrichOrder(orderID, samplePlannerEnrichmentRequest()); err != nil {
		t.Fatalf("Expected enrichment to succeed, got %v", err)
	}

	relResp := generator.CreateMockOrderStatusResponse(orderID)
	if len(relResp.D.ToMaintenanceOrderOperation.Results) != 2 {
		t.Fatalf("Expected 2 enriched operations, got %d", len(relResp.D.ToMaintenanceOrderOperation.Results))
	}
	if relResp.D.ToMaintenanceOrderOperation.Results[0].OperationStatus != "REL" {
		t.Errorf("Expected operation status REL before TECO, got %s", relResp.D.ToMaintenanceOrderOperation.Results[0].OperationStatus)
	}
	if relResp.D.ToMaintenanceOrderOperation.Results[0].ActualWorkQuantity != "" {
		t.Errorf("Expected actual work to be hidden before TECO, got %s", relResp.D.ToMaintenanceOrderOperation.Results[0].ActualWorkQuantity)
	}

	generator.mu.Lock()
	*generator.orderStore[orderID].EnrichedAt = time.Now().Add(-31 * time.Second)
	generator.mu.Unlock()

	tecoResp := generator.CreateMockOrderStatusResponse(orderID)
	firstOp := tecoResp.D.ToMaintenanceOrderOperation.Results[0]
	if firstOp.OperationText != "Disassemble pump" {
		t.Errorf("Expected enriched operation text, got %s", firstOp.OperationText)
	}
	if firstOp.OperationStatus != "CNF" {
		t.Errorf("Expected operation status CNF at TECO, got %s", firstOp.OperationStatus)
	}
	if firstOp.ActualWorkQuantity != "4.5" {
		t.Errorf("Expected actual work 4.5, got %s", firstOp.ActualWorkQuantity)
	}
	if firstOp.OpActualExecutionStartDateTime != "2025-10-21T08:00:00Z" {
		t.Errorf("Expected actual start timestamp from enrichment, got %s", firstOp.OpActualExecutionStartDateTime)
	}
	if firstOp.OpActualExecutionEndDateTime != "2025-10-21T12:30:00Z" {
		t.Errorf("Expected actual end timestamp from enrichment, got %s", firstOp.OpActualExecutionEndDateTime)
	}
	if len(firstOp.ToMaintOrderOpComponent2.Results) != 1 {
		t.Fatalf("Expected one planner component on first operation, got %d", len(firstOp.ToMaintOrderOpComponent2.Results))
	}
	component := firstOp.ToMaintOrderOpComponent2.Results[0]
	if component.Product != "SEAL-X200" {
		t.Errorf("Expected planner material SEAL-X200, got %s", component.Product)
	}
	if component.MaintOrdOpCompRequiredQuantity != "1.250" {
		t.Errorf("Expected used quantity to overwrite visible component quantity, got %s", component.MaintOrdOpCompRequiredQuantity)
	}
	if !component.ReservationIsFinallyIssued {
		t.Error("Expected final issue to be true")
	}
}

func samplePlannerEnrichmentRequest() *models.PlannerEnrichmentRequest {
	finalIssue := true
	return &models.PlannerEnrichmentRequest{
		PlannedStartDateTime: "2025-10-21T08:00:00Z",
		PlannedEndDateTime:   "2025-10-21T16:00:00Z",
		MainWorkCenter:       "MECH-01",
		MainWorkCenterPlant:  "1000",
		Operations: []models.EnrichedOperation{
			{
				Operation:           "0010",
				ControlKey:          "PM01",
				Description:         "Disassemble pump",
				WorkCenter:          "MECH-01",
				Plant:               "1000",
				PlannedWorkQuantity: "4.0",
				ActualWorkQuantity:  "4.5",
				WorkQuantityUnit:    "H",
				ActualStartDateTime: "2025-10-21T08:00:00Z",
				ActualEndDateTime:   "2025-10-21T12:30:00Z",
				Components: []models.EnrichedComponent{
					{
						Component:         "0001",
						Material:          "SEAL-X200",
						Description:       "Pump seal",
						RequiredQuantity:  "1.000",
						UsedQuantity:      "1.250",
						Unit:              "EA",
						Plant:             "1000",
						StorageLocation:   "0001",
						GoodsMovementType: "261",
						FinalIssue:        &finalIssue,
					},
				},
			},
			{
				Operation:           "0020",
				Description:         "Replace seal",
				WorkCenter:          "MECH-01",
				Plant:               "1000",
				PlannedWorkQuantity: "3.0",
				WorkQuantityUnit:    "H",
			},
		},
	}
}
