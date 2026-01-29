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
			MaintenanceOrderType:   "PM01",
			Description:            "Test order",
			Equipment:              "10000045",
			FunctionalLocation:     "TEST-LOC",
			Plant:                  "1000",
			MaintenanceNotification: "200000123",
		}
		createResp := generator.CreateMockOrderResponse(req)
		orderID := createResp.D.MaintenanceOrder

		// Immediately after creation - should be CRTD
		statusResp := generator.CreateMockOrderStatusResponse(orderID)
		if statusResp.D.OrderStatus != "CRTD" {
			t.Errorf("Expected status CRTD immediately after creation, got %s", statusResp.D.OrderStatus)
		}

		// Wait 11 seconds - should be REL
		time.Sleep(11 * time.Second)
		statusResp = generator.CreateMockOrderStatusResponse(orderID)
		if statusResp.D.OrderStatus != "REL" {
			t.Errorf("Expected status REL after 11 seconds, got %s", statusResp.D.OrderStatus)
		}

		// Wait another 20 seconds (31 total) - should be TECO
		time.Sleep(20 * time.Second)
		statusResp = generator.CreateMockOrderStatusResponse(orderID)
		if statusResp.D.OrderStatus != "TECO" {
			t.Errorf("Expected status TECO after 31 seconds, got %s", statusResp.D.OrderStatus)
		}
	})
}
