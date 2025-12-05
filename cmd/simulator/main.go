package main

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"

	"sap-adaptor/internal/models"
	"sap-adaptor/internal/sap"

	"github.com/sirupsen/logrus"
)

var (
	logger        *logrus.Logger
	mockGenerator *sap.MockGenerator
)

func main() {
	// Initialize logger
	logger = logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Initialize mock generator
	mockGenerator = sap.NewMockGenerator()

	// Get port from environment or use default
	port := os.Getenv("SAP_SIMULATOR_PORT")
	if port == "" {
		port = "8081"
	}

	// Setup HTTP routes
	http.HandleFunc("/API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification", handleCreateNotification)
	http.HandleFunc("/API_MAINTENANCE_ORDER/", handleMaintenanceOrder)
	http.HandleFunc("/health", handleHealth)

	logger.WithField("port", port).Info("🚀 SAP Simulator starting")
	logger.Info("Available endpoints:")
	logger.Info("  POST /API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification")
	logger.Info("  POST /API_MAINTENANCE_ORDER/A_MaintenanceOrder")
	logger.Info("  GET  /API_MAINTENANCE_ORDER/A_MaintenanceOrder('...')")
	logger.Info("  GET  /health")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.WithError(err).Fatal("Failed to start server")
	}
}

// handleCreateNotification handles notification creation requests
func handleCreateNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger.Info("📥 Received notification creation request")

	// Parse request body
	var req models.SAPNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithError(err).Error("Failed to parse request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.WithFields(logrus.Fields{
		"equipment": req.Equipment,
		"plant":     req.Plant,
		"description": req.Description,
	}).Info("Creating mock notification")

	// Generate mock response
	resp := mockGenerator.CreateMockNotificationResponse(&req)

	logger.WithField("notificationId", resp.D.Notification).Info("✅ Mock notification created")

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// handleMaintenanceOrder handles both POST (create) and GET (retrieve) for maintenance orders
func handleMaintenanceOrder(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleCreateOrder(w, r)
	case http.MethodGet:
		handleGetOrder(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreateOrder handles order creation requests
func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	logger.Info("📥 Received order creation request")

	// Parse request body
	var req models.SAPOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithError(err).Error("Failed to parse request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.WithFields(logrus.Fields{
		"equipment":     req.Equipment,
		"plant":         req.Plant,
		"notification":  req.MaintenanceNotification,
		"operations":    len(req.ToMaintenanceOrderOperation),
	}).Info("Creating mock order")

	// Generate mock response
	resp := mockGenerator.CreateMockOrderResponse(&req)

	logger.WithFields(logrus.Fields{
		"orderId": resp.D.MaintenanceOrder,
		"status":  resp.D.OrderStatus,
	}).Info("✅ Mock order created")

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// handleGetOrder handles order retrieval requests
func handleGetOrder(w http.ResponseWriter, r *http.Request) {
	// Extract order ID from URL
	// Expected format: /API_MAINTENANCE_ORDER/A_MaintenanceOrder('400000123')
	orderID := extractOrderID(r.URL.Path)
	if orderID == "" {
		logger.Error("Failed to extract order ID from URL")
		http.Error(w, "Invalid order ID format", http.StatusBadRequest)
		return
	}

	logger.WithField("orderId", orderID).Info("📥 Received order retrieval request")

	// Generate mock response
	resp := mockGenerator.CreateMockOrderStatusResponse(orderID)

	logger.WithFields(logrus.Fields{
		"orderId": resp.D.MaintenanceOrder,
		"status":  resp.D.OrderStatus,
	}).Info("✅ Mock order retrieved")

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "SAP Simulator",
		"version": "1.0.0",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// extractOrderID extracts the order ID from the URL path
// Expected format: /API_MAINTENANCE_ORDER/A_MaintenanceOrder('400000123')
func extractOrderID(path string) string {
	// Remove query parameters
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// Try to match the pattern A_MaintenanceOrder('ID')
	re := regexp.MustCompile(`A_MaintenanceOrder\('([^']+)'\)`)
	matches := re.FindStringSubmatch(path)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback: try to extract anything after A_MaintenanceOrder/
	if idx := strings.Index(path, "A_MaintenanceOrder/"); idx != -1 {
		return strings.TrimPrefix(path[idx:], "A_MaintenanceOrder/")
	}

	return ""
}
