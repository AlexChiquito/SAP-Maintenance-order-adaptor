package handlers

import (
	"net/http"

	"sap-adaptor/internal/models"
	"sap-adaptor/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// MaintenanceHandler handles HTTP requests for maintenance operations
type MaintenanceHandler struct {
	maintenanceService *services.MaintenanceService
	logger            *logrus.Logger
	validator         *validator.Validate
}

// NewMaintenanceHandler creates a new maintenance handler
func NewMaintenanceHandler(maintenanceService *services.MaintenanceService, logger *logrus.Logger) *MaintenanceHandler {
	return &MaintenanceHandler{
		maintenanceService: maintenanceService,
		logger:            logger,
		validator:         validator.New(),
	}
}

// CreateMaintenanceOrder handles POST /maintenance-orders
// @Summary Create Maintenance Order Event
// @Description Creates a maintenance order in SAP based on equipment information from Digital Twin
// @Tags Maintenance Orders
// @Accept json
// @Produce json
// @Param request body models.MaintenanceOrderEvent true "Maintenance Order Event"
// @Success 201 {object} models.MaintenanceOrderResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /maintenance-orders [post]
func (h *MaintenanceHandler) CreateMaintenanceOrder(c *gin.Context) {
	var event models.MaintenanceOrderEvent

	// Bind and validate request
	if err := c.ShouldBindJSON(&event); err != nil {
		h.logger.WithError(err).Error("Failed to bind JSON request")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request format",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Validate the request
	if err := h.validator.Struct(&event); err != nil {
		h.logger.WithError(err).Error("Request validation failed")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Validation failed",
			Code:    "VALIDATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Process the maintenance order event
	response, err := h.maintenanceService.ProcessMaintenanceOrderEvent(c.Request.Context(), &event)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process maintenance order event")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create maintenance order",
			Code:    "PROCESSING_ERROR",
			Details: err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orderId":        response.OrderID,
		"notificationId": response.NotificationID,
		"status":         response.Status,
	}).Info("Maintenance order created successfully")

	c.JSON(http.StatusCreated, response)
}

// GetMaintenanceOrder handles GET /maintenance-orders/:id
// @Summary Get Maintenance Order Status
// @Description Retrieves the current status and details of a maintenance order
// @Tags Maintenance Orders
// @Produce json
// @Param id path string true "Maintenance Order ID"
// @Success 200 {object} models.MaintenanceOrderStatus
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /maintenance-orders/{id} [get]
func (h *MaintenanceHandler) GetMaintenanceOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Order ID is required",
			Code:  "MISSING_ORDER_ID",
		})
		return
	}

	// Get maintenance order status
	status, err := h.maintenanceService.GetMaintenanceOrderStatus(c.Request.Context(), orderID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"orderId": orderID,
			"error":   err,
		}).Error("Failed to get maintenance order status")

		// Check if it's a not found error
		if err.Error() == "SAP API returned status 404" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "Maintenance order not found",
				Code:  "ORDER_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to retrieve maintenance order",
			Code:    "RETRIEVAL_ERROR",
			Details: err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orderId": status.OrderID,
		"status":  status.Status,
	}).Info("Maintenance order status retrieved successfully")

	c.JSON(http.StatusOK, status)
}

// HandleMaintenanceDone handles POST /maintenance-done
// @Summary Handle Maintenance Done Event
// @Description Receives maintenance completion notification from SAP and forwards it to Digital Twin
// @Tags Maintenance Events
// @Accept json
// @Produce json
// @Param request body models.MaintenanceDoneEvent true "Maintenance Done Event"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /maintenance-done [post]
func (h *MaintenanceHandler) HandleMaintenanceDone(c *gin.Context) {
	var event models.MaintenanceDoneEvent

	// Bind and validate request
	if err := c.ShouldBindJSON(&event); err != nil {
		h.logger.WithError(err).Error("Failed to bind JSON request")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request format",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Validate the request
	if err := h.validator.Struct(&event); err != nil {
		h.logger.WithError(err).Error("Request validation failed")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Validation failed",
			Code:    "VALIDATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Process the maintenance done event
	err := h.maintenanceService.HandleMaintenanceDoneEvent(c.Request.Context(), &event)
	if err != nil {
		h.logger.WithError(err).Error("Failed to handle maintenance done event")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to process maintenance done event",
			Code:    "PROCESSING_ERROR",
			Details: err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orderId": event.OrderID,
		"status":  event.Status,
	}).Info("Maintenance done event processed successfully")

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Maintenance done event processed successfully",
	})
}

// HealthCheck handles GET /health
// @Summary Health Check
// @Description Check if the service is running
// @Tags System
// @Produce json
// @Success 200 {object} models.SuccessResponse
// @Router /health [get]
func (h *MaintenanceHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "SAP Adaptor is running",
	})
}

// GetMetrics handles GET /metrics (placeholder for future metrics implementation)
// @Summary Get Service Metrics
// @Description Get service performance metrics
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /metrics [get]
func (h *MaintenanceHandler) GetMetrics(c *gin.Context) {
	// Placeholder for metrics - in a real implementation, you would collect
	// metrics about orders created, processing times, error rates, etc.
	metrics := map[string]interface{}{
		"service":     "sap-adaptor",
		"version":     "1.0.0",
		"uptime":      "running",
		"orders_created": 0, // This would be tracked in a real implementation
		"errors_total":   0, // This would be tracked in a real implementation
	}

	c.JSON(http.StatusOK, metrics)
}
