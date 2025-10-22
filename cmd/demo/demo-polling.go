package main

import (
	"context"
	"fmt"
	"sap-adaptor/internal/config"
	"sap-adaptor/internal/models"
	"sap-adaptor/internal/services"
	"sap-adaptor/internal/sap"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("=== SAP Adaptor Polling Demo ===")
	fmt.Println("This demonstrates how SAP Adaptor polls SAP for status changes")
	fmt.Println("and detects when an order reaches TECO status.")
	fmt.Println()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create config with simulator mode
	cfg := config.SAPConfig{
		BaseURL:       "simulator",
		SimulatorMode: true,
		Timeout:       30,
	}

	// Create SAP client and service
	sapClient := sap.NewClient(cfg, logger)
	maintenanceService := services.NewMaintenanceService(sapClient, logger)

	// Create a test order first
	fmt.Println("1. Creating a test order...")
	digitalTwinEvent := &models.MaintenanceOrderEvent{
		EquipmentID:          "10000045",
		FunctionalLocation:   "FL100-200-300",
		Plant:                "1000",
		Description:          "Test order for polling demo",
		Priority:             "3",
		MaintenanceOrderType: "PM01",
		PlannedStartTime:     &[]time.Time{time.Now().Add(1 * time.Hour)}[0],
		PlannedEndTime:       &[]time.Time{time.Now().Add(9 * time.Hour)}[0],
		Operations: []models.MaintenanceOperation{
			{
				Text:          "Test operation",
				WorkCenter:    "TEST-WC01",
				Duration:      4.0,
				DurationUnit: "H",
			},
		},
	}

	// Process the order
	response, err := maintenanceService.ProcessMaintenanceOrderEvent(context.Background(), digitalTwinEvent)
	if err != nil {
		fmt.Printf("Error creating order: %v\n", err)
		return
	}

	fmt.Printf("✅ Order created: %s\n", response.OrderID)
	fmt.Printf("   Notification: %s\n", response.NotificationID)
	fmt.Printf("   Status: %s\n", response.Status)
	fmt.Println()

	// Now demonstrate polling
	fmt.Println("2. Starting status monitoring (polling every 30 seconds)...")
	fmt.Println("   This simulates how SAP Adaptor would monitor for TECO status")
	fmt.Println("   In simulator mode, status changes based on order ID digits")
	fmt.Println()

	// Create a callback function that would notify Digital Twin
	callback := func(status *models.MaintenanceOrderStatus) error {
		fmt.Println("🎉 TECO DETECTED! Order completed!")
		fmt.Printf("   Order ID: %s\n", status.OrderID)
		fmt.Printf("   Status: %s\n", status.Status)
		fmt.Printf("   Equipment: %s\n", status.EquipmentID)
		fmt.Printf("   Plant: %s\n", status.Plant)
		fmt.Println()
		fmt.Println("📤 This is where SAP Adaptor would send notification to Digital Twin:")
		fmt.Printf("   POST /api/v1/maintenance-completed\n")
		fmt.Printf("   {\n")
		fmt.Printf("     \"orderId\": \"%s\",\n", status.OrderID)
		fmt.Printf("     \"status\": \"%s\",\n", status.Status)
		fmt.Printf("     \"equipmentId\": \"%s\",\n", status.EquipmentID)
		fmt.Printf("     \"plant\": \"%s\",\n", status.Plant)
		fmt.Printf("     \"completedAt\": \"%s\"\n", time.Now().Format(time.RFC3339))
		fmt.Printf("   }\n")
		return nil
	}

	// Start monitoring (with a timeout for demo)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Println("⏰ Monitoring for 2 minutes (or until TECO detected)...")
	fmt.Println("   Polling SAP every 30 seconds...")
	fmt.Println()

	err = maintenanceService.MonitorOrderStatus(ctx, response.OrderID, callback)
	if err != nil {
		if err == context.DeadlineExceeded {
			fmt.Println("⏰ Demo timeout reached - monitoring stopped")
		} else {
			fmt.Printf("❌ Monitoring error: %v\n", err)
		}
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("In production, this polling would continue until:")
	fmt.Println("- Order reaches TECO/CLSD status")
	fmt.Println("- Order is cancelled")
	fmt.Println("- System is shut down")
}

