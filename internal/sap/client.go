package sap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"sap-adaptor/internal/config"
	"sap-adaptor/internal/models"

	"github.com/sirupsen/logrus"
)

// Client represents the SAP API client
type Client struct {
	config     config.SAPConfig
	httpClient *http.Client
	logger     *logrus.Logger
	simulatorMode bool
}

// NewClient creates a new SAP client
func NewClient(cfg config.SAPConfig, logger *logrus.Logger) *Client {
	simulatorMode := cfg.SimulatorMode || cfg.BaseURL == "" || cfg.BaseURL == "simulator"
	
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		logger: logger,
		simulatorMode: simulatorMode,
	}
}

// CreateNotification creates a maintenance notification in SAP
func (c *Client) CreateNotification(ctx context.Context, req *models.SAPNotificationRequest) (*models.SAPNotificationResponse, error) {
	c.logger.WithFields(logrus.Fields{
		"equipment": req.Equipment,
		"plant":     req.Plant,
		"simulatorMode": c.simulatorMode,
	}).Info("Creating SAP maintenance notification")

	// If in simulator mode, return mock response
	if c.simulatorMode {
		c.logger.Info("Running in simulator mode - returning mock notification response")
		return c.createMockNotificationResponse(req), nil
	}

	// Prepare request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", 
		c.config.BaseURL+"/API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification", 
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (no authentication in simulator mode)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		c.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(respBody),
		}).Error("SAP notification creation failed")
		return nil, fmt.Errorf("SAP API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var notificationResp models.SAPNotificationResponse
	if err := json.Unmarshal(respBody, &notificationResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"notificationId": notificationResp.D.Notification,
	}).Info("SAP maintenance notification created successfully")

	return &notificationResp, nil
}

// CreateOrder creates a maintenance order in SAP
func (c *Client) CreateOrder(ctx context.Context, req *models.SAPOrderRequest) (*models.SAPOrderResponse, error) {
	c.logger.WithFields(logrus.Fields{
		"equipment": req.Equipment,
		"plant":     req.Plant,
		"notification": req.MaintenanceNotification,
		"simulatorMode": c.simulatorMode,
	}).Info("Creating SAP maintenance order")

	// If in simulator mode, return mock response
	if c.simulatorMode {
		c.logger.Info("Running in simulator mode - returning mock order response")
		return c.createMockOrderResponse(req), nil
	}

	// Prepare request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", 
		c.config.BaseURL+"/API_MAINTENANCE_ORDER/A_MaintenanceOrder", 
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (no authentication in simulator mode)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		c.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(respBody),
		}).Error("SAP order creation failed")
		return nil, fmt.Errorf("SAP API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var orderResp models.SAPOrderResponse
	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"orderId": orderResp.D.MaintenanceOrder,
	}).Info("SAP maintenance order created successfully")

	return &orderResp, nil
}

// GetOrder retrieves a maintenance order from SAP
func (c *Client) GetOrder(ctx context.Context, orderID string) (*models.SAPOrderResponse, error) {
	c.logger.WithFields(logrus.Fields{
		"orderId": orderID,
		"simulatorMode": c.simulatorMode,
	}).Info("Retrieving SAP maintenance order")

	// If in simulator mode, return mock response
	if c.simulatorMode {
		c.logger.Info("Running in simulator mode - returning mock order status response")
		return c.createMockOrderStatusResponse(orderID), nil
	}

	// Create URL with expand parameter
	baseURL := c.config.BaseURL + "/API_MAINTENANCE_ORDER/A_MaintenanceOrder('" + orderID + "')"
	params := url.Values{}
	params.Add("$expand", "to_MaintenanceOrderOperation")
	fullURL := baseURL + "?" + params.Encode()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (no authentication in simulator mode)
	httpReq.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(respBody),
		}).Error("SAP order retrieval failed")
		return nil, fmt.Errorf("SAP API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var orderResp models.SAPOrderResponse
	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"orderId": orderResp.D.MaintenanceOrder,
		"status":  orderResp.D.OrderStatus,
	}).Info("SAP maintenance order retrieved successfully")

	return &orderResp, nil
}

// createMockNotificationResponse creates a mock notification response for simulator mode
func (c *Client) createMockNotificationResponse(req *models.SAPNotificationRequest) *models.SAPNotificationResponse {
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

// createMockOrderResponse creates a mock order response for simulator mode
func (c *Client) createMockOrderResponse(req *models.SAPOrderRequest) *models.SAPOrderResponse {
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

// createMockOrderStatusResponse creates a mock order status response for simulator mode
func (c *Client) createMockOrderStatusResponse(orderID string) *models.SAPOrderResponse {
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

// ConvertMaintenanceOrderEventToNotificationRequest converts a MaintenanceOrderEvent to SAP notification request
func ConvertMaintenanceOrderEventToNotificationRequest(event *models.MaintenanceOrderEvent) *models.SAPNotificationRequest {
	return &models.SAPNotificationRequest{
		NotificationType:   "M1", // Default notification type
		Description:        event.Description,
		Equipment:          event.EquipmentID,
		FunctionalLocation: event.FunctionalLocation,
		Plant:              event.Plant,
		Priority:           event.Priority,
	}
}

// ConvertMaintenanceOrderEventToOrderRequest converts a MaintenanceOrderEvent to SAP order request
func ConvertMaintenanceOrderEventToOrderRequest(event *models.MaintenanceOrderEvent, notificationID string) *models.SAPOrderRequest {
	req := &models.SAPOrderRequest{
		MaintenanceOrderType:    event.MaintenanceOrderType,
		Description:             event.Description,
		Equipment:               event.EquipmentID,
		FunctionalLocation:      event.FunctionalLocation,
		Plant:                   event.Plant,
		MaintenancePlanningPlant: event.Plant, // Default to same plant
		Priority:                event.Priority,
		MaintenanceNotification: notificationID,
	}

	// Add time fields if provided
	if event.PlannedStartTime != nil {
		req.MaintOrdBasicStartDateTime = event.PlannedStartTime.Format(time.RFC3339)
	}
	if event.PlannedEndTime != nil {
		req.MaintOrdBasicEndDateTime = event.PlannedEndTime.Format(time.RFC3339)
	}

	// Convert operations
	for _, op := range event.Operations {
		sapOp := models.SAPOrderOperation{
			OperationText:             op.Text,
			WorkCenter:                op.WorkCenter,
			Plant:                     event.Plant,
			OperationControlKey:       event.MaintenanceOrderType,
			OperationStandardDuration: strconv.FormatFloat(op.Duration, 'f', -1, 64),
			OperationDurationUnit:     op.DurationUnit,
		}
		req.ToMaintenanceOrderOperation = append(req.ToMaintenanceOrderOperation, sapOp)
	}

	return req
}

// ConvertSAPOrderResponseToStatus converts SAP order response to MaintenanceOrderStatus
func ConvertSAPOrderResponseToStatus(resp *models.SAPOrderResponse) *models.MaintenanceOrderStatus {
	status := &models.MaintenanceOrderStatus{
		OrderID:        resp.D.MaintenanceOrder,
		Status:         resp.D.OrderStatus,
		Description:    resp.D.Description,
		EquipmentID:    resp.D.Equipment,
		Plant:          resp.D.Plant,
		NotificationID: resp.D.MaintenanceNotification,
	}

	// Parse time fields if provided
	if resp.D.MaintOrdBasicStartDateTime != "" {
		if t, err := time.Parse(time.RFC3339, resp.D.MaintOrdBasicStartDateTime); err == nil {
			status.ActualStartTime = &t
		}
	}
	if resp.D.MaintOrdBasicEndDateTime != "" {
		if t, err := time.Parse(time.RFC3339, resp.D.MaintOrdBasicEndDateTime); err == nil {
			status.ActualEndTime = &t
		}
	}

	// Convert operations
	for _, op := range resp.D.ToMaintenanceOrderOperation.Results {
		opStatus := models.OperationStatus{
			OperationID:      op.MaintenanceOrderOperation,
			Text:             op.OperationText,
			Status:           op.OperationStatus,
			WorkQuantityUnit: op.WorkQuantityUnit,
		}
		if op.ActualWorkQuantity != "" {
			if qty, err := strconv.ParseFloat(op.ActualWorkQuantity, 64); err == nil {
				opStatus.ActualWorkQuantity = qty
			}
		}
		status.Operations = append(status.Operations, opStatus)
	}

	return status
}
