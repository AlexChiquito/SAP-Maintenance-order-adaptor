package models

import (
	"testing"
	"time"
)

func TestMaintenanceOrderEventValidation(t *testing.T) {
	// Test valid maintenance order event
	event := &MaintenanceOrderEvent{
		EquipmentID:          "10000045",
		FunctionalLocation:   "FL100-200-300",
		Plant:                "1000",
		Description:          "Test maintenance order",
		Priority:             "3",
		MaintenanceOrderType: "PM01",
		PlannedStartTime:     &[]time.Time{time.Now()}[0],
		PlannedEndTime:       &[]time.Time{time.Now().Add(8 * time.Hour)}[0],
		Operations: []MaintenanceOperation{
			{
				Text:         "Test operation",
				WorkCenter:   "TEST-WC01",
				Duration:     4.0,
				DurationUnit: "H",
			},
		},
	}

	// Test conversion to SAP notification request
	notificationReq := ConvertMaintenanceOrderEventToNotificationRequest(event)
	if notificationReq.Equipment != event.EquipmentID {
		t.Errorf("Expected equipment %s, got %s", event.EquipmentID, notificationReq.Equipment)
	}
	if notificationReq.Description != event.Description {
		t.Errorf("Expected description %s, got %s", event.Description, notificationReq.Description)
	}

	// Test conversion to SAP order request
	orderReq := ConvertMaintenanceOrderEventToOrderRequest(event, "200000123")
	if orderReq.Equipment != event.EquipmentID {
		t.Errorf("Expected equipment %s, got %s", event.EquipmentID, orderReq.Equipment)
	}
	if orderReq.MaintenanceNotification != "200000123" {
		t.Errorf("Expected notification ID 200000123, got %s", orderReq.MaintenanceNotification)
	}
}

func TestMaintenanceDoneEventValidation(t *testing.T) {
	// Test valid maintenance done event
	event := &MaintenanceDoneEvent{
		OrderID:         "400000789",
		Status:          "TECO",
		CompletedAt:     &[]time.Time{time.Now()}[0],
		ActualWorkHours: 6.0,
		Notes:           "Maintenance completed successfully",
	}

	if event.OrderID == "" {
		t.Error("OrderID should not be empty")
	}
	if event.Status == "" {
		t.Error("Status should not be empty")
	}
}
