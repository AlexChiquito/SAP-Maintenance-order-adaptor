package sap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sap-adaptor/internal/config"
	"sap-adaptor/internal/models"

	"github.com/sirupsen/logrus"
)

func TestClient_CreateNotification(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification" {
			t.Errorf("Expected path /API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse request body
		var req models.SAPNotificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Create response
		resp := models.SAPNotificationResponse{
			D: struct {
				Notification string `json:"Notification"`
				Description  string `json:"Description"`
				Plant        string `json:"Plant"`
			}{
				Notification: "200000123",
				Description:  req.Description,
				Plant:        req.Plant,
			},
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	logger := logrus.New()
	cfg := config.SAPConfig{
		BaseURL: server.URL,
		Timeout: 30,
	}
	client := NewClient(cfg, logger)

	// Create notification
	req := &models.SAPNotificationRequest{
		NotificationType:   "M1",
		Description:        "Test notification",
		Equipment:          "10000045",
		FunctionalLocation: "FL100-200-300",
		Plant:              "1000",
		Priority:           "3",
	}

	resp, err := client.CreateNotification(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	if resp.D.Notification != "200000123" {
		t.Errorf("Expected notification ID 200000123, got %s", resp.D.Notification)
	}
}

func TestClient_CreateOrder(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/API_MAINTENANCE_ORDER/A_MaintenanceOrder" {
			t.Errorf("Expected path /API_MAINTENANCE_ORDER/A_MaintenanceOrder, got %s", r.URL.Path)
		}

		// Parse request body
		var req models.SAPOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Create response
		resp := models.SAPOrderResponse{}
		resp.D.MaintenanceOrder = "400000123"
		resp.D.MaintenanceOrderType = req.MaintenanceOrderType
		resp.D.Description = req.Description
		resp.D.Equipment = req.Equipment
		resp.D.Plant = req.Plant
		resp.D.OrderStatus = "CRTD"
		resp.D.MaintenanceNotification = req.MaintenanceNotification
		resp.D.ToMaintenanceOrderOperation.Results = []models.SAPOrderOperationResponse{}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	logger := logrus.New()
	cfg := config.SAPConfig{
		BaseURL: server.URL,
		Timeout: 30,
	}
	client := NewClient(cfg, logger)

	// Create order
	req := &models.SAPOrderRequest{
		MaintenanceOrderType:    "PM01",
		Description:             "Test order",
		Equipment:               "10000045",
		Plant:                   "1000",
		MaintenanceNotification: "200000123",
		Priority:                "3",
	}

	resp, err := client.CreateOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if resp.D.MaintenanceOrder != "400000123" {
		t.Errorf("Expected order ID 400000123, got %s", resp.D.MaintenanceOrder)
	}
	if resp.D.OrderStatus != "CRTD" {
		t.Errorf("Expected status CRTD, got %s", resp.D.OrderStatus)
	}
}

func TestClient_GetOrder(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Create response
		resp := models.SAPOrderResponse{}
		resp.D.MaintenanceOrder = "400000123"
		resp.D.MaintenanceOrderType = "PM01"
		resp.D.Description = "Test order"
		resp.D.Equipment = "10000045"
		resp.D.Plant = "1000"
		resp.D.OrderStatus = "TECO"
		resp.D.MaintenanceNotification = "200000123"
		resp.D.ToMaintenanceOrderOperation.Results = []models.SAPOrderOperationResponse{}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	logger := logrus.New()
	cfg := config.SAPConfig{
		BaseURL: server.URL,
		Timeout: 30,
	}
	client := NewClient(cfg, logger)

	// Get order
	resp, err := client.GetOrder(context.Background(), "400000123")
	if err != nil {
		t.Fatalf("GetOrder failed: %v", err)
	}

	if resp.D.MaintenanceOrder != "400000123" {
		t.Errorf("Expected order ID 400000123, got %s", resp.D.MaintenanceOrder)
	}
	if resp.D.OrderStatus != "TECO" {
		t.Errorf("Expected status TECO, got %s", resp.D.OrderStatus)
	}
}

func TestConvertMaintenanceOrderEventToNotificationRequest(t *testing.T) {
	event := &models.MaintenanceOrderEvent{
		EquipmentID:          "10000045",
		FunctionalLocation:   "FL100-200-300",
		Plant:                "1000",
		Description:          "Test maintenance",
		Priority:             "3",
		MaintenanceOrderType: "PM01",
	}

	req := ConvertMaintenanceOrderEventToNotificationRequest(event)

	if req.Equipment != event.EquipmentID {
		t.Errorf("Expected equipment %s, got %s", event.EquipmentID, req.Equipment)
	}
	if req.FunctionalLocation != event.FunctionalLocation {
		t.Errorf("Expected functional location %s, got %s", event.FunctionalLocation, req.FunctionalLocation)
	}
	if req.Plant != event.Plant {
		t.Errorf("Expected plant %s, got %s", event.Plant, req.Plant)
	}
	if req.Description != event.Description {
		t.Errorf("Expected description %s, got %s", event.Description, req.Description)
	}
	if req.Priority != event.Priority {
		t.Errorf("Expected priority %s, got %s", event.Priority, req.Priority)
	}
}

func TestConvertMaintenanceOrderEventToOrderRequest(t *testing.T) {
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

	notificationID := "200000123"
	req := ConvertMaintenanceOrderEventToOrderRequest(event, notificationID)

	if req.Equipment != event.EquipmentID {
		t.Errorf("Expected equipment %s, got %s", event.EquipmentID, req.Equipment)
	}
	if req.MaintenanceNotification != notificationID {
		t.Errorf("Expected notification ID %s, got %s", notificationID, req.MaintenanceNotification)
	}
	if req.MaintenanceOrderType != event.MaintenanceOrderType {
		t.Errorf("Expected order type %s, got %s", event.MaintenanceOrderType, req.MaintenanceOrderType)
	}
	if len(req.ToMaintenanceOrderOperation) != len(event.Operations) {
		t.Errorf("Expected %d operations, got %d", len(event.Operations), len(req.ToMaintenanceOrderOperation))
	}
}
