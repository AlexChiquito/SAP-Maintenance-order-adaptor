package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sap-adaptor/internal/config"
	"sap-adaptor/internal/models"
	"sap-adaptor/internal/sap"

	"github.com/sirupsen/logrus"
)

func TestMaintenanceService_ProcessMaintenanceOrderEvent(t *testing.T) {
	// Create a mock SAP server
	var notificationID string
	var orderID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification":
			// Create notification
			notificationID = "200000123"
			resp := models.SAPNotificationResponse{
				D: struct {
					Notification string `json:"Notification"`
					Description  string `json:"Description"`
					Plant        string `json:"Plant"`
				}{
					Notification: notificationID,
					Description:  "Test notification",
					Plant:        "1000",
				},
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)

		case "/API_MAINTENANCE_ORDER/A_MaintenanceOrder":
			// Create order
			orderID = "400000789"
			resp := models.SAPOrderResponse{}
			resp.D.MaintenanceOrder = orderID
			resp.D.MaintenanceOrderType = "PM01"
			resp.D.Description = "Test order"
			resp.D.Equipment = "10000045"
			resp.D.Plant = "1000"
			resp.D.OrderStatus = "CRTD"
			resp.D.MaintenanceNotification = notificationID
			resp.D.ToMaintenanceOrderOperation.Results = []models.SAPOrderOperationResponse{}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)

		default:
			// Get order (verification)
			resp := models.SAPOrderResponse{}
			resp.D.MaintenanceOrder = orderID
			resp.D.MaintenanceOrderType = "PM01"
			resp.D.Description = "Test order"
			resp.D.Equipment = "10000045"
			resp.D.Plant = "1000"
			resp.D.OrderStatus = "CRTD"
			resp.D.MaintenanceNotification = notificationID
			resp.D.ToMaintenanceOrderOperation.Results = []models.SAPOrderOperationResponse{}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	// Create service
	logger := logrus.New()
	cfg := config.SAPConfig{
		BaseURL: server.URL,
		Timeout: 30,
	}
	sapClient := sap.NewClient(cfg, logger)
	service := NewMaintenanceService(sapClient, logger)

	// Create test event
	startTime := time.Now()
	endTime := startTime.Add(8 * time.Hour)
	event := &models.MaintenanceOrderEvent{
		EquipmentID:          "10000045",
		FunctionalLocation:   "FL100-200-300",
		Plant:                "1000",
		Description:          "Test maintenance",
		Priority:             "3",
		MaintenanceOrderType: "PM01",
		PlannedStartTime:     &startTime,
		PlannedEndTime:       &endTime,
		Operations: []models.MaintenanceOperation{
			{
				Text:         "Test operation",
				WorkCenter:   "TEST-WC01",
				Duration:     4.0,
				DurationUnit: "H",
			},
		},
	}

	// Process event
	resp, err := service.ProcessMaintenanceOrderEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("ProcessMaintenanceOrderEvent failed: %v", err)
	}

	// Verify response
	if resp.MaintenanceOrder != orderID {
		t.Errorf("Expected order ID %s, got %s", orderID, resp.MaintenanceOrder)
	}
	if resp.MaintenanceNotification != notificationID {
		t.Errorf("Expected notification ID %s, got %s", notificationID, resp.MaintenanceNotification)
	}
	if resp.Status != "CRTD" {
		t.Errorf("Expected status CRTD, got %s", resp.Status)
	}
	if resp.Message != "Maintenance order created successfully" {
		t.Errorf("Expected success message, got %s", resp.Message)
	}
}

func TestMaintenanceService_GetMaintenanceOrderStatus(t *testing.T) {
	// Create a mock SAP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.SAPOrderResponse{}
		resp.D.MaintenanceOrder = "400000789"
		resp.D.MaintenanceOrderType = "PM01"
		resp.D.Description = "Test order"
		resp.D.Equipment = "10000045"
		resp.D.Plant = "1000"
		resp.D.OrderStatus = "TECO"
		resp.D.MaintenanceNotification = "200000123"
		resp.D.ToMaintenanceOrderOperation.Results = []models.SAPOrderOperationResponse{
			{
				MaintenanceOrder:          "400000789",
				MaintenanceOrderOperation: "0010",
				OperationText:             "Test operation",
				WorkCenter:                "TEST-WC01",
				OperationStatus:           "CNF",
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create service
	logger := logrus.New()
	cfg := config.SAPConfig{
		BaseURL: server.URL,
		Timeout: 30,
	}
	sapClient := sap.NewClient(cfg, logger)
	service := NewMaintenanceService(sapClient, logger)

	// Get order status
	status, err := service.GetMaintenanceOrderStatus(context.Background(), "400000789")
	if err != nil {
		t.Fatalf("GetMaintenanceOrderStatus failed: %v", err)
	}

	// Verify status
	if status.MaintenanceOrder != "400000789" {
		t.Errorf("Expected order ID 400000789, got %s", status.MaintenanceOrder)
	}
	if status.Status != "TECO" {
		t.Errorf("Expected status TECO, got %s", status.Status)
	}
	if len(status.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(status.Operations))
	}
	if status.Operations[0].Status != "CNF" {
		t.Errorf("Expected operation status CNF, got %s", status.Operations[0].Status)
	}
}
