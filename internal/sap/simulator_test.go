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

	tests := []struct {
		orderID        string
		expectedStatus string
	}{
		{"4000000", "CRTD"},
		{"4000001", "CRTD"},
		{"4000002", "CRTD"},
		{"4000003", "REL"},
		{"4000004", "REL"},
		{"4000005", "REL"},
		{"4000006", "TECO"},
		{"4000007", "TECO"},
		{"4000008", "TECO"},
		{"4000009", "CLSD"},
	}

	for _, tt := range tests {
		t.Run(tt.orderID, func(t *testing.T) {
			resp := generator.CreateMockOrderStatusResponse(tt.orderID)

			if resp == nil {
				t.Fatal("Expected response, got nil")
			}

			if resp.D.MaintenanceOrder != tt.orderID {
				t.Errorf("Expected order ID %s, got %s", tt.orderID, resp.D.MaintenanceOrder)
			}

			if resp.D.OrderStatus != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, resp.D.OrderStatus)
			}
		})
	}
}
