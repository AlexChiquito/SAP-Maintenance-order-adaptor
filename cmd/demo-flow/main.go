package main

import (
	"context"
	"fmt"
	"time"

	"sap-adaptor/internal/config"
	"sap-adaptor/internal/models"
	"sap-adaptor/internal/sap"
	"sap-adaptor/internal/services"

	"github.com/sirupsen/logrus"
)

func main() {
	printBanner()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Check if simulator is running
	fmt.Println("📡 Checking if SAP Simulator is running on http://localhost:8081...")
	cfg := config.SAPConfig{
		BaseURL: "http://localhost:8081",
		Timeout: 30,
	}

	sapClient := sap.NewClient(cfg, logger)
	maintenanceService := services.NewMaintenanceService(sapClient, logger)

	// Step 1: Show the incoming event from Digital Twin
	printStep(1, "Digital Twin Event Generator sends Maintenance Order Event")
	digitalTwinEvent := &models.MaintenanceOrderEvent{
		EquipmentID:          "10000045",
		FunctionalLocation:   "FL100-200-300",
		Plant:                "1000",
		Description:          "Pump maintenance - bearing replacement",
		Priority:             "3",
		MaintenanceOrderType: "PM01",
		PlannedStartTime:     &[]time.Time{time.Now().Add(1 * time.Hour)}[0],
		PlannedEndTime:       &[]time.Time{time.Now().Add(9 * time.Hour)}[0],
		Operations: []models.MaintenanceOperation{
			{
				Text:         "Replace bearings",
				WorkCenter:   "MAINT-01",
				Duration:     4.0,
				DurationUnit: "H",
			},
			{
				Text:         "Lubricate pump",
				WorkCenter:   "MAINT-01",
				Duration:     1.0,
				DurationUnit: "H",
			},
		},
	}

	printEventData("Incoming Event", digitalTwinEvent)
	pauseForEffect()

	// Step 2: SAP Adaptor processes the event
	printStep(2, "SAP Adaptor processes the event")
	printSubStep("2.1", "Creating SAP Maintenance Notification")
	printHTTPCall("POST", "http://localhost:8081/API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification")

	response, err := maintenanceService.ProcessMaintenanceOrderEvent(context.Background(), digitalTwinEvent)
	if err != nil {
		printError("Failed to process event", err)
		printHint()
		return
	}

	printSuccess("Notification Created", response.NotificationID)
	pauseForEffect()

	printSubStep("2.2", "Creating SAP Maintenance Order")
	printHTTPCall("POST", "http://localhost:8081/API_MAINTENANCE_ORDER/A_MaintenanceOrder")
	printSuccess("Order Created", response.OrderID)
	printInfo("Status", response.Status)
	pauseForEffect()

	printSubStep("2.3", "Verifying Order Creation")
	printHTTPCall("GET", fmt.Sprintf("http://localhost:8081/API_MAINTENANCE_ORDER/A_MaintenanceOrder('%s')", response.OrderID))
	printSuccess("Order Verified", response.OrderID)
	pauseForEffect()

	// Step 3: Show status monitoring
	printStep(3, "SAP Adaptor monitors order status")
	fmt.Println("   🔄 Polling SAP every 30 seconds to detect completion...")
	fmt.Println()

	// Poll a few times to show the monitoring
	for i := 1; i <= 3; i++ {
		printInfo(fmt.Sprintf("Poll #%d", i), "Checking order status...")
		printHTTPCall("GET", fmt.Sprintf("http://localhost:8081/API_MAINTENANCE_ORDER/A_MaintenanceOrder('%s')", response.OrderID))

		status, err := maintenanceService.GetMaintenanceOrderStatus(context.Background(), response.OrderID)
		if err != nil {
			printError("Failed to get status", err)
			return
		}

		printInfo("Current Status", status.Status)
		
		if status.Status == "TECO" || status.Status == "CLSD" {
			printSuccess("Order Completed!", status.OrderID)
			printStep(4, "SAP Adaptor sends completion event to Digital Twin")
			printHTTPCall("POST", "http://digital-twin/api/v1/maintenance-completed")
			printCompletionEvent(status)
			break
		}

		if i < 3 {
			fmt.Println("   ⏳ Status not TECO yet, continuing to monitor...")
			pauseForEffect()
		}
	}

	printSummary(response.OrderID, response.NotificationID)
}

func printBanner() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         SAP Adaptor - Data Flow Demonstration                ║")
	fmt.Println("║    Showing HTTP communication between systems                ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func printStep(num int, description string) {
	fmt.Printf("\n┌─────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│ STEP %d: %s\n", num, description)
	fmt.Printf("└─────────────────────────────────────────────────────────────┘\n\n")
}

func printSubStep(num, description string) {
	fmt.Printf("  %s %s\n", num, description)
}

func printHTTPCall(method, url string) {
	fmt.Printf("     → %s %s\n", method, url)
}

func printSuccess(label, value string) {
	fmt.Printf("     ✅ %s: %s\n", label, value)
}

func printInfo(label, value string) {
	fmt.Printf("     ℹ️  %s: %s\n", label, value)
}

func printError(message string, err error) {
	fmt.Printf("     ❌ %s: %v\n", message, err)
}

func printEventData(title string, event *models.MaintenanceOrderEvent) {
	fmt.Printf("   📋 %s:\n", title)
	fmt.Printf("      Equipment ID: %s\n", event.EquipmentID)
	fmt.Printf("      Plant: %s\n", event.Plant)
	fmt.Printf("      Description: %s\n", event.Description)
	fmt.Printf("      Priority: %s\n", event.Priority)
	fmt.Printf("      Operations: %d\n", len(event.Operations))
	for i, op := range event.Operations {
		fmt.Printf("        %d. %s (%.1f %s)\n", i+1, op.Text, op.Duration, op.DurationUnit)
	}
	fmt.Println()
}

func printCompletionEvent(status *models.MaintenanceOrderStatus) {
	fmt.Println("   📤 Completion Event Payload:")
	fmt.Println("   {")
	fmt.Printf("     \"orderId\": \"%s\",\n", status.OrderID)
	fmt.Printf("     \"status\": \"%s\",\n", status.Status)
	fmt.Printf("     \"equipmentId\": \"%s\",\n", status.EquipmentID)
	fmt.Printf("     \"plant\": \"%s\",\n", status.Plant)
	fmt.Printf("     \"completedAt\": \"%s\",\n", time.Now().Format(time.RFC3339))
	fmt.Printf("     \"operations\": %d\n", len(status.Operations))
	fmt.Println("   }")
	fmt.Println()
}

func printSummary(orderID, notificationID string) {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     DEMO SUMMARY                              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("  🔄 Data Flow Completed:")
	fmt.Println()
	fmt.Println("  1️⃣  Digital Twin → SAP Adaptor")
	fmt.Println("      Maintenance Order Event (JSON)")
	fmt.Println()
	fmt.Println("  2️⃣  SAP Adaptor → SAP Simulator (HTTP)")
	fmt.Println("      ├─ POST Notification")
	fmt.Println("      ├─ POST Order")
	fmt.Println("      └─ GET Order Status")
	fmt.Println()
	fmt.Println("  3️⃣  SAP Adaptor → Digital Twin")
	fmt.Println("      Maintenance Completed Event (JSON)")
	fmt.Println()
	fmt.Printf("  📝 Order ID: %s\n", orderID)
	fmt.Printf("  📝 Notification ID: %s\n", notificationID)
	fmt.Println()
	fmt.Println("  ✨ Key Changes in New Architecture:")
	fmt.Println("     • Simulator runs as separate HTTP service (port 8081)")
	fmt.Println("     • Real HTTP requests between adaptor and simulator")
	fmt.Println("     • Can be deployed in separate containers")
	fmt.Println("     • Easy to swap simulator with real SAP")
	fmt.Println()
}

func printHint() {
	fmt.Println("\n💡 HINT: Make sure the SAP Simulator is running first:")
	fmt.Println("   Terminal 1: cd /path/to/project && ./bin/simulator")
	fmt.Println("   Terminal 2: cd /path/to/project && ./bin/demo")
	fmt.Println()
	fmt.Println("   Or use Docker Compose:")
	fmt.Println("   docker compose up")
	fmt.Println()
}

func pauseForEffect() {
	time.Sleep(800 * time.Millisecond)
}
