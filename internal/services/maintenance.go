package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"sap-adaptor/internal/models"
	"sap-adaptor/internal/sap"

	"github.com/sirupsen/logrus"
)

// MaintenanceService handles maintenance order business logic
type MaintenanceService struct {
	sapClient        *sap.Client
	logger           *logrus.Logger
	digitalTwinURL   string
	digitalTwinAPIKey string
}

// NewMaintenanceService creates a new maintenance service
func NewMaintenanceService(sapClient *sap.Client, logger *logrus.Logger) *MaintenanceService {
	return &MaintenanceService{
		sapClient:        sapClient,
		logger:           logger,
		digitalTwinURL:   "",  // Will be set from config
		digitalTwinAPIKey: "", // Will be set from config
	}
}

// SetDigitalTwinConfig sets the Digital Twin configuration
func (s *MaintenanceService) SetDigitalTwinConfig(baseURL, apiKey string) {
	s.digitalTwinURL = baseURL
	s.digitalTwinAPIKey = apiKey
}

// ProcessMaintenanceOrderEvent processes a maintenance order event following the SAP integration workflow
func (s *MaintenanceService) ProcessMaintenanceOrderEvent(ctx context.Context, event *models.MaintenanceOrderEvent) (*models.MaintenanceOrderResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"equipmentId": event.EquipmentID,
		"plant":       event.Plant,
		"description": event.Description,
	}).Info("Processing maintenance order event")

	// Step 1: Create SAP Maintenance Notification
	s.logger.Info("Step 1: Creating SAP maintenance notification")
	notificationReq := sap.ConvertMaintenanceOrderEventToNotificationRequest(event)
	notificationResp, err := s.sapClient.CreateNotification(ctx, notificationReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create SAP notification: %w", err)
	}

	notificationID := notificationResp.D.Notification
	s.logger.WithField("notificationId", notificationID).Info("SAP notification created successfully")

	// Step 2: Create SAP Maintenance Order with notification reference
	s.logger.Info("Step 2: Creating SAP maintenance order")
	orderReq := sap.ConvertMaintenanceOrderEventToOrderRequest(event, notificationID)
	orderResp, err := s.sapClient.CreateOrder(ctx, orderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create SAP order: %w", err)
	}

	orderID := orderResp.D.MaintenanceOrder
	s.logger.WithField("orderId", orderID).Info("SAP maintenance order created successfully")

	// Step 3: Verify order was created successfully
	s.logger.Info("Step 3: Verifying order creation")
	verifyResp, err := s.sapClient.GetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify order creation: %w", err)
	}

	if verifyResp.D.MaintenanceOrder != orderID {
		return nil, fmt.Errorf("order verification failed: expected %s, got %s", orderID, verifyResp.D.MaintenanceOrder)
	}

	s.logger.WithFields(logrus.Fields{
		"orderId":        orderID,
		"notificationId": notificationID,
		"status":         verifyResp.D.OrderStatus,
	}).Info("Order verification completed successfully")

	// Return success response
	response := &models.MaintenanceOrderResponse{
		MaintenanceOrder:       orderID,
		MaintenanceNotification: notificationID,
		Status:                  verifyResp.D.OrderStatus,
		Message:                 "Maintenance order created successfully",
		CreatedAt:               time.Now(),
	}

	// Start background polling to monitor order status
	go func() {
		s.logger.WithField("orderId", orderID).Info("Starting background monitoring for order")
		
		// Create a context for monitoring (will run indefinitely until TECO/CLSD or error)
		monitorCtx := context.Background()
		
		err := s.MonitorOrderStatus(monitorCtx, orderID, func(status *models.MaintenanceOrderStatus) error {
			return s.notifyDigitalTwin(status)
		})
		
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"orderId": orderID,
				"error":   err,
			}).Error("Order monitoring failed")
		}
	}()

	return response, nil
}

// GetMaintenanceOrderStatus retrieves the current status of a maintenance order
func (s *MaintenanceService) GetMaintenanceOrderStatus(ctx context.Context, orderID string) (*models.MaintenanceOrderStatus, error) {
	s.logger.WithField("orderId", orderID).Info("Retrieving maintenance order status")

	// Get order from SAP
	orderResp, err := s.sapClient.GetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order from SAP: %w", err)
	}

	// Convert to status model
	status := sap.ConvertSAPOrderResponseToStatus(orderResp)

	s.logger.WithFields(logrus.Fields{
		"maintenanceOrder": status.MaintenanceOrder,
		"status":           status.Status,
	}).Info("Maintenance order status retrieved successfully")

	return status, nil
}

// HandleMaintenanceDoneEvent processes a maintenance done event from SAP
func (s *MaintenanceService) HandleMaintenanceDoneEvent(ctx context.Context, event *models.MaintenanceDoneEvent) error {
	s.logger.WithFields(logrus.Fields{
		"orderId": event.OrderID,
		"status":  event.Status,
	}).Info("Processing maintenance done event")

	// Verify the order exists and get its details
	orderStatus, err := s.GetMaintenanceOrderStatus(ctx, event.OrderID)
	if err != nil {
		return fmt.Errorf("failed to verify order: %w", err)
	}

	// Log the completion
	s.logger.WithFields(logrus.Fields{
		"orderId":        event.OrderID,
		"status":         event.Status,
		"completedAt":   event.CompletedAt,
		"actualWorkHours": event.ActualWorkHours,
		"notes":          event.Notes,
		"equipment":      orderStatus.Equipment,
		"plant":          orderStatus.Plant,
	}).Info("Maintenance completed successfully")

	// TODO: Here you would typically send a notification back to the Digital Twin system
	// For now, we'll just log the completion
	s.logger.Info("Maintenance done event processed successfully")

	return nil
}

// MonitorOrderStatus monitors an order until completion (for background processing)
func (s *MaintenanceService) MonitorOrderStatus(ctx context.Context, orderID string, callback func(*models.MaintenanceOrderStatus) error) error {
	s.logger.WithField("orderId", orderID).Info("Starting order status monitoring")

	// Check immediately first, then use ticker
	checkStatus := func() error {
		status, err := s.GetMaintenanceOrderStatus(ctx, orderID)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"orderId": orderID,
				"error":   err,
			}).Error("Failed to get order status during monitoring")
			return err
		}

		// Check if order is completed
		if status.Status == "TECO" || status.Status == "CLSD" {
			s.logger.WithFields(logrus.Fields{
				"orderId": orderID,
				"status":  status.Status,
			}).Info("Order completed, stopping monitoring")

			// Call the callback function
			if callback != nil {
				if err := callback(status); err != nil {
					s.logger.WithFields(logrus.Fields{
						"orderId": orderID,
						"error":   err,
					}).Error("Callback function failed")
					return fmt.Errorf("callback failed: %w", err)
				}
			}

			return nil
		}

		s.logger.WithFields(logrus.Fields{
			"orderId": orderID,
			"status":  status.Status,
		}).Info("Order still in progress, continuing monitoring")
		
		return fmt.Errorf("order not completed yet")
	}

	// Check immediately - if no error returned, order is completed
	if err := checkStatus(); err == nil {
		return nil
	}

	// Continue with periodic checks
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.WithField("orderId", orderID).Info("Order monitoring cancelled")
			return ctx.Err()
		case <-ticker.C:
			if err := checkStatus(); err == nil {
				return nil
			}
		}
	}
}

// notifyDigitalTwin sends a completion notification to the Digital Twin system
func (s *MaintenanceService) notifyDigitalTwin(status *models.MaintenanceOrderStatus) error {
	if s.digitalTwinURL == "" {
		s.logger.Warn("Digital Twin URL not configured, skipping notification")
		return nil
	}

	s.logger.WithFields(logrus.Fields{
		"maintenanceOrder": status.MaintenanceOrder,
		"status":           status.Status,
	}).Info("Sending completion notification to Digital Twin")

	// Build notification payload
	notification := map[string]interface{}{
		"maintenanceOrder":       status.MaintenanceOrder,
		"status":                 status.Status,
		"description":            status.Description,
		"equipment":              status.Equipment,
		"plant":                  status.Plant,
		"maintenanceNotification": status.MaintenanceNotification,
		"completedAt":            time.Now().Format(time.RFC3339),
		"operations":             status.Operations,

		"objectList":             status.ObjectList,
	}

	// Marshal to JSON
	payload, err := json.Marshal(notification)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal notification payload")
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Build HTTP request
	endpoint := s.digitalTwinURL + "/api/v1/maintenance-completed"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		s.logger.WithError(err).Error("Failed to create HTTP request")
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if s.digitalTwinAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.digitalTwinAPIKey)
	}

	// Send HTTP request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		s.logger.WithError(err).WithField("endpoint", endpoint).Error("Failed to send notification to Digital Twin")
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.WithFields(logrus.Fields{
			"statusCode": resp.StatusCode,
			"response":   string(body),
			"endpoint":   endpoint,
		}).Warn("Digital Twin returned non-success status")
		return fmt.Errorf("digital twin returned status %d: %s", resp.StatusCode, string(body))
	}

	s.logger.WithFields(logrus.Fields{
		"endpoint":   endpoint,
		"statusCode": resp.StatusCode,
		"maintenanceOrder": status.MaintenanceOrder,
	}).Info("Successfully notified Digital Twin")

	return nil
}
