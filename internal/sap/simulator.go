package sap

import (
	"fmt"
	"time"

	"sap-adaptor/internal/models"
)

// MockGenerator provides functions to generate mock SAP responses
type MockGenerator struct{}

// NewMockGenerator creates a new mock generator
func NewMockGenerator() *MockGenerator {
	return &MockGenerator{}
}

// CreateMockNotificationResponse creates a mock notification response for simulator mode
func (g *MockGenerator) CreateMockNotificationResponse(req *models.SAPNotificationRequest) *models.SAPNotificationResponse {
	// Generate a mock notification ID
	notificationID := fmt.Sprintf("200000%03d", time.Now().Unix()%1000)
	
	return &models.SAPNotificationResponse{
		D: struct {
			Notification   string `json:"Notification"`
			Description    string `json:"Description"`
			Plant          string `json:"Plant"`
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
			MaintenanceOrder:                orderID,
			MaintenanceOrderOperation:       operationID,
			OperationText:                   op.OperationText,
			WorkCenter:                      op.WorkCenter,
			OperationControlKey:             op.OperationControlKey,
			OperationStandardDuration:       op.OperationStandardDuration,
			OperationDurationUnit:           op.OperationDurationUnit,
			Metadata: struct {
				ID  string `json:"id"`
				URI string `json:"uri"`
				Type string `json:"type"`
			}{
				ID:   fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='%s')", orderID, operationID),
				URI:  fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='%s')", orderID, operationID),
				Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderOperationType",
			},
		})
	}
	
	return &models.SAPOrderResponse{
		D: struct {
			MaintenanceOrder                string `json:"MaintenanceOrder"`
			MaintenanceOrderType            string `json:"MaintenanceOrderType"`
			Description                     string `json:"Description"`
			Equipment                       string `json:"Equipment"`
			Plant                           string `json:"Plant"`
			OrderStatus                     string `json:"OrderStatus"`
			MaintOrdBasicStartDateTime      string `json:"MaintOrdBasicStartDateTime"`
			MaintOrdBasicEndDateTime        string `json:"MaintOrdBasicEndDateTime"`
			MaintenanceNotification         string `json:"MaintenanceNotification"`
			Metadata                        struct {
				ID  string `json:"id"`
				URI string `json:"uri"`
				Type string `json:"type"`
			} `json:"__metadata"`
			ToMaintenanceOrderOperation struct {
				Results []models.SAPOrderOperationResponse `json:"results"`
			} `json:"to_MaintenanceOrderOperation"`
		}{
			MaintenanceOrder:                orderID,
			MaintenanceOrderType:            req.MaintenanceOrderType,
			Description:                     req.Description,
			Equipment:                       req.Equipment,
			Plant:                           req.Plant,
			OrderStatus:                     "CRTD", // Created status
			MaintOrdBasicStartDateTime:      req.MaintOrdBasicStartDateTime,
			MaintOrdBasicEndDateTime:        req.MaintOrdBasicEndDateTime,
			MaintenanceNotification:         req.MaintenanceNotification,
			Metadata: struct {
				ID  string `json:"id"`
				URI string `json:"uri"`
				Type string `json:"type"`
			}{
				ID:   fmt.Sprintf(".../A_MaintenanceOrder('%s')", orderID),
				URI:  fmt.Sprintf(".../A_MaintenanceOrder('%s')", orderID),
				Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderType",
			},
			ToMaintenanceOrderOperation: struct {
				Results []models.SAPOrderOperationResponse `json:"results"`
			}{
				Results: operations,
			},
		},
	}
}

// CreateMockOrderStatusResponse creates a mock order status response for simulator mode
// The status is determined by the last digit of the order ID to simulate progression:
// - 0-2: CRTD (Created)
// - 3-5: REL (Released)
// - 6-8: TECO (Technically Completed)
// - 9: CLSD (Closed)
func (g *MockGenerator) CreateMockOrderStatusResponse(orderID string) *models.SAPOrderResponse {
	// Simulate different statuses based on order ID
	status := "CRTD" // Default to created
	if len(orderID) > 0 {
		// Simple logic to simulate different statuses
		lastDigit := orderID[len(orderID)-1]
		switch lastDigit {
		case '0', '1', '2':
			status = "CRTD" // Created
		case '3', '4', '5':
			status = "REL"  // Released
		case '6', '7', '8':
			status = "TECO" // Technically completed
		case '9':
			status = "CLSD" // Closed
		}
	}
	
	return &models.SAPOrderResponse{
		D: struct {
			MaintenanceOrder                string `json:"MaintenanceOrder"`
			MaintenanceOrderType            string `json:"MaintenanceOrderType"`
			Description                     string `json:"Description"`
			Equipment                       string `json:"Equipment"`
			Plant                           string `json:"Plant"`
			OrderStatus                     string `json:"OrderStatus"`
			MaintOrdBasicStartDateTime      string `json:"MaintOrdBasicStartDateTime"`
			MaintOrdBasicEndDateTime        string `json:"MaintOrdBasicEndDateTime"`
			MaintenanceNotification         string `json:"MaintenanceNotification"`
			Metadata                        struct {
				ID  string `json:"id"`
				URI string `json:"uri"`
				Type string `json:"type"`
			} `json:"__metadata"`
			ToMaintenanceOrderOperation struct {
				Results []models.SAPOrderOperationResponse `json:"results"`
			} `json:"to_MaintenanceOrderOperation"`
		}{
			MaintenanceOrder:                orderID,
			MaintenanceOrderType:            "PM01",
			Description:                     "Mock maintenance order",
			Equipment:                       "10000045",
			Plant:                           "1000",
			OrderStatus:                     status,
			MaintOrdBasicStartDateTime:      time.Now().Format(time.RFC3339),
			MaintOrdBasicEndDateTime:        time.Now().Add(8 * time.Hour).Format(time.RFC3339),
			MaintenanceNotification:         "200000123",
			Metadata: struct {
				ID  string `json:"id"`
				URI string `json:"uri"`
				Type string `json:"type"`
			}{
				ID:   fmt.Sprintf(".../A_MaintenanceOrder('%s')", orderID),
				URI:  fmt.Sprintf(".../A_MaintenanceOrder('%s')", orderID),
				Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderType",
			},
			ToMaintenanceOrderOperation: struct {
				Results []models.SAPOrderOperationResponse `json:"results"`
			}{
				Results: []models.SAPOrderOperationResponse{
					{
						MaintenanceOrder:                orderID,
						MaintenanceOrderOperation:       "0010",
						OperationText:                   "Mock operation",
						WorkCenter:                      "MOCK-WC01",
						OperationControlKey:             "PM01",
						OperationStandardDuration:       "4",
						OperationDurationUnit:           "H",
						OperationStatus:                 "CNF",
						ActualWorkQuantity:              "4.0",
						WorkQuantityUnit:                "H",
						Metadata: struct {
							ID  string `json:"id"`
							URI string `json:"uri"`
							Type string `json:"type"`
						}{
							ID:   fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='0010')", orderID),
							URI:  fmt.Sprintf(".../A_MaintenanceOrderOperation(MaintenanceOrder='%s',MaintenanceOrderOperation='0010')", orderID),
							Type: "API_MAINTENANCE_ORDER.A_MaintenanceOrderOperationType",
						},
					},
				},
			},
		},
	}
}
