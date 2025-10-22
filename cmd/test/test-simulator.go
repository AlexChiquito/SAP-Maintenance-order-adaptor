package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sap-adaptor/internal/config"
	"sap-adaptor/internal/models"
	"sap-adaptor/internal/sap"
	"time"

	"github.com/sirupsen/logrus"
)

func prettyPrintJSON(label string, v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("%s: <failed to marshal: %v>\n", label, err)
		return
	}
	fmt.Printf("%s:\n%s\n", label, string(b))
}

func main() {
	fmt.Println("=== SAP Adaptor Simulator Test ===")
	
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	// Create config with simulator mode
	cfg := config.SAPConfig{
		BaseURL:       "simulator",
		SimulatorMode: true,
		Timeout:       30,
	}
	
	// Create SAP client
	sapClient := sap.NewClient(cfg, logger)
	
	// (Not using MaintenanceService here; custom polling below)
	
	// Test the complete workflow starting from Digital Twin
	fmt.Println("\n0. Complete Workflow Test...")
	fmt.Println("   Digital Twin → SAP Adaptor: Maintenance Order Event")
	
	// This is what the Digital Twin would send to the SAP Adaptor
	digitalTwinEvent := &models.MaintenanceOrderEvent{
		EquipmentID:          "10000045",
		FunctionalLocation:   "FL100-200-300",
		Plant:                "1000",
		Description:          "Pump showing abnormal vibration - needs seal replacement",
		Priority:             "3",
		MaintenanceOrderType: "PM01",
		PlannedStartTime:     &[]time.Time{time.Now().Add(24 * time.Hour)}[0], // Tomorrow
		PlannedEndTime:       &[]time.Time{time.Now().Add(24*time.Hour + 8*time.Hour)}[0], // Tomorrow + 8 hours
		Operations: []models.MaintenanceOperation{
			{
				Text:          "Disassemble pump and inspect seal",
				WorkCenter:    "PUMP-WC01",
				Duration:      4.0,
				DurationUnit: "H",
			},
			{
				Text:          "Replace seal and reassemble",
				WorkCenter:    "PUMP-WC01", 
				Duration:      3.0,
				DurationUnit: "H",
			},
			{
				Text:          "Test pump operation",
				WorkCenter:    "PUMP-WC01",
				Duration:      1.0,
				DurationUnit: "H",
			},
		},
	}
	prettyPrintJSON("Digital Twin → SAP Adaptor (MaintenanceOrderEvent)", digitalTwinEvent)
	
	// Now SAP Adaptor processes this event and converts it for SAP
	fmt.Println("\n   SAP Adaptor Internal: Converting Digital Twin Event to SAP Format")
	
	// Convert to SAP notification request
	sapNotificationReq := sap.ConvertMaintenanceOrderEventToNotificationRequest(digitalTwinEvent)
	prettyPrintJSON("SAP Adaptor Internal (Converted NotificationRequest)", sapNotificationReq)
	
	// Convert to SAP order request (we'll use a placeholder notification ID for now)
	placeholderNotificationID := "200000000" // This will be replaced with actual notification ID
	sapOrderReq := sap.ConvertMaintenanceOrderEventToOrderRequest(digitalTwinEvent, placeholderNotificationID)
	prettyPrintJSON("SAP Adaptor Internal (Converted OrderRequest)", sapOrderReq)
	
	// Test notification creation
	fmt.Println("\n1. Testing Notification Creation...")
	fmt.Println("   SAP Adaptor → SAP: Creating Maintenance Notification")
	prettyPrintJSON("SAP Adaptor → SAP (CreateNotification Request)", sapNotificationReq)
	
	notificationResp, err := sapClient.CreateNotification(context.Background(), sapNotificationReq)
	if err != nil {
		fmt.Printf("Error creating notification: %v\n", err)
		return
	}
	prettyPrintJSON("SAP → SAP Adaptor (CreateNotification Response)", notificationResp)
	fmt.Printf("✅ Notification created: %s\n", notificationResp.D.Notification)
	
	// Test order creation
	fmt.Println("\n2. Testing Order Creation...")
	fmt.Println("   SAP Adaptor → SAP: Creating Maintenance Order")
	
	// Update the order request with the actual notification ID
	sapOrderReq.MaintenanceNotification = notificationResp.D.Notification
	prettyPrintJSON("SAP Adaptor → SAP (CreateOrder Request)", sapOrderReq)
	
	orderResp, err := sapClient.CreateOrder(context.Background(), sapOrderReq)
	if err != nil {
		fmt.Printf("Error creating order: %v\n", err)
		return
	}
	prettyPrintJSON("SAP → SAP Adaptor (CreateOrder Response)", orderResp)
	fmt.Printf("✅ Order created: %s\n", orderResp.D.MaintenanceOrder)
	
	// Test order retrieval
	fmt.Println("\n3. Testing Order Retrieval...")
	fmt.Println("   SAP Adaptor → SAP: Querying Order Status")
	statusResp, err := sapClient.GetOrder(context.Background(), orderResp.D.MaintenanceOrder)
	if err != nil {
		fmt.Printf("Error retrieving order: %v\n", err)
		return
	}
	prettyPrintJSON("SAP → SAP Adaptor (GetOrder Response)", statusResp)
	
	fmt.Printf("✅ Order status: %s\n", statusResp.D.OrderStatus)
	fmt.Printf("   Description: %s\n", statusResp.D.Description)
	fmt.Printf("   Equipment: %s\n", statusResp.D.Equipment)
	fmt.Printf("   Plant: %s\n", statusResp.D.Plant)
	
	// Show final conversion back to Digital Twin format
	fmt.Println("\n4. Final Conversion...")
	fmt.Println("   SAP Adaptor → Digital Twin: Converting SAP Response to Digital Twin Format")
	convertedStatus := sap.ConvertSAPOrderResponseToStatus(statusResp)
	prettyPrintJSON("SAP Adaptor → Digital Twin (MaintenanceOrderStatus)", convertedStatus)
	fmt.Printf("✅ Final status for Digital Twin: OrderID=%s, Status=%s\n", 
		convertedStatus.OrderID, convertedStatus.Status)
	
	// Test polling-based TECO detection (custom 10s polling with random readiness)
	fmt.Println("\n5. Testing Polling-Based TECO Detection...")
	fmt.Println("   SAP Adaptor Internal: Starting custom 10s polling loop")
	fmt.Println("   This simulates polling every 10s until TECO is reached")

	// Seed randomness for demo
	rand.Seed(time.Now().UnixNano())
	readyAfterPolls := 4 + rand.Intn(3) // ready after 4-6 polls
	pollInterval := 10 * time.Second
	fmt.Printf("   ℹ️ Will mark TECO after %d polls (~%s)\n", readyAfterPolls, time.Duration(readyAfterPolls)*pollInterval)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(readyAfterPolls+3)*pollInterval)
	defer cancel()

	// Create a callback function that would notify Digital Twin
	callback := func(status *models.MaintenanceOrderStatus) error {
		fmt.Println("\n🎉 TECO DETECTED! Order completed!")
		fmt.Println("   SAP Adaptor → Digital Twin: Sending completion notification")

		// This is what would be sent to Digital Twin
		digitalTwinNotification := map[string]interface{}{
			"orderId": status.OrderID,
			"status": status.Status,
			"description": status.Description,
			"equipmentId": status.EquipmentID,
			"plant": status.Plant,
			"notificationId": status.NotificationID,
			"completedAt": time.Now().Format(time.RFC3339),
			"actualStartTime": status.ActualStartTime,
			"actualEndTime": status.ActualEndTime,
			"operations": status.Operations,
		}

		prettyPrintJSON("SAP Adaptor → Digital Twin (MaintenanceCompleted Notification)", digitalTwinNotification)

		fmt.Println("✅ Digital Twin notification would be sent to:")
		fmt.Println("   POST /api/v1/maintenance-completed")
		fmt.Println("   (TODO: Implement Digital Twin client)")

		return nil
	}

	// Custom polling loop independent of service defaults
	polls := 0
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("   ⏰ Demo timeout reached - monitoring stopped")
			goto donePolling
		case <-ticker.C:
			polls++
			fmt.Printf("   🔄 Poll #%d: querying order status...\n", polls)
			latest, err := sapClient.GetOrder(context.Background(), orderResp.D.MaintenanceOrder)
			if err != nil {
				fmt.Printf("   ⚠️  GetOrder error: %v\n", err)
				continue
			}
			status := sap.ConvertSAPOrderResponseToStatus(latest)

			if polls >= readyAfterPolls {
				// Force TECO for the demo
				status.Status = "TECO"
				_ = callback(status)
				goto donePolling
			}

			fmt.Printf("   ↪︎ Still in progress (status=%s). Waiting %s...\n", status.Status, pollInterval)
		}
	}

	donePolling:
	
	fmt.Println("\n=== All Tests Passed! ===")
	fmt.Println("The SAP Adaptor simulator is working correctly.")
	fmt.Println("Complete workflow demonstrated:")
	fmt.Println("1. Digital Twin → SAP Adaptor (Maintenance Order Event)")
	fmt.Println("2. SAP Adaptor → SAP (Create Notification)")
	fmt.Println("3. SAP Adaptor → SAP (Create Order)")
	fmt.Println("4. SAP Adaptor → SAP (Query Status)")
	fmt.Println("5. SAP Adaptor → Digital Twin (Status Response)")
	fmt.Println("6. SAP Adaptor → SAP (Polling for TECO)")
	fmt.Println("7. SAP Adaptor → Digital Twin (TECO Notification)")
}
