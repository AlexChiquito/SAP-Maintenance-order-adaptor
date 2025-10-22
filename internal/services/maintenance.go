package services

import (
	"context"
	"fmt"
	"time"

	"sap-adaptor/internal/models"
	"sap-adaptor/internal/sap"

	"github.com/sirupsen/logrus"
)

// MaintenanceService handles maintenance order business logic
type MaintenanceService struct {
	sapClient *sap.Client
	logger    *logrus.Logger
}

// NewMaintenanceService creates a new maintenance service
func NewMaintenanceService(sapClient *sap.Client, logger *logrus.Logger) *MaintenanceService {
	return &MaintenanceService{
		sapClient: sapClient,
		logger:    logger,
	}
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
		OrderID:        orderID,
		NotificationID: notificationID,
		Status:         verifyResp.D.OrderStatus,
		Message:        "Maintenance order created successfully",
		CreatedAt:      time.Now(),
	}

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
		"orderId": status.OrderID,
		"status":  status.Status,
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
		"equipmentId":    orderStatus.EquipmentID,
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

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.WithField("orderId", orderID).Info("Order monitoring cancelled")
			return ctx.Err()
		case <-ticker.C:
			status, err := s.GetMaintenanceOrderStatus(ctx, orderID)
			if err != nil {
				s.logger.WithFields(logrus.Fields{
					"orderId": orderID,
					"error":   err,
				}).Error("Failed to get order status during monitoring")
				continue
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
						return err
					}
				}

				return nil
			}

			s.logger.WithFields(logrus.Fields{
				"orderId": orderID,
				"status":  status.Status,
			}).Debug("Order still in progress, continuing monitoring")
		}
	}
}

