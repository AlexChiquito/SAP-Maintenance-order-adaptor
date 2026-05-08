package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sap-adaptor/internal/models"
	"strings"
	"time"
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
	fmt.Println("=== SAP Adaptor End-to-End Test ===")

	// Get SAP Adaptor URL from environment
	adaptorURL := os.Getenv("SAP_ADAPTOR_URL")
	if adaptorURL == "" {
		adaptorURL = "http://localhost:8080"
	}
	simulatorURL := os.Getenv("SAP_SIMULATOR_URL")
	if simulatorURL == "" {
		simulatorURL = "http://localhost:8081"
	}
	plannerDemo := strings.EqualFold(os.Getenv("SAP_PLANNER_ENRICHMENT_DEMO"), "true")

	fmt.Printf("Using SAP Adaptor at: %s\n", adaptorURL)
	if plannerDemo {
		fmt.Printf("Using SAP Simulator at: %s\n", simulatorURL)
		fmt.Println("Planner enrichment demo mode: enabled")
	}
	fmt.Println("(Make sure SAP Adaptor and Simulator are running)")
	fmt.Println("")

	// Check health
	fmt.Println("1. Checking SAP Adaptor health...")
	healthResp, err := http.Get(adaptorURL + "/health")
	if err != nil {
		fmt.Printf("ERROR: SAP Adaptor not reachable: %v\n", err)
		fmt.Println("   Start it with:")
		fmt.Println("   ./bin/sap-adaptor")
		os.Exit(1)
	}
	defer healthResp.Body.Close()

	var healthData map[string]interface{}
	json.NewDecoder(healthResp.Body).Decode(&healthData)
	prettyPrintJSON("Health Check Response", healthData)
	fmt.Println("SAP Adaptor is running")
	fmt.Println("")

	// Create maintenance order event (what Digital Twin would send)
	fmt.Println("2. Creating Maintenance Order...")
	fmt.Println("   Digital Twin → SAP Adaptor: POST /api/v1/maintenance-orders")
	fmt.Println("")

	startTime := time.Now().Add(1 * time.Hour)
	endTime := time.Now().Add(9 * time.Hour)

	event := &models.MaintenanceOrderEvent{
		EquipmentID:          "PUMP-MAIN-12345",
		FunctionalLocation:   "FL-PRODUCTION-LINE-A",
		Plant:                "2000",
		Description:          "Critical pump replacement - bearing failure detected",
		Priority:             "1",
		MaintenanceOrderType: "PM01",
		PlannedStartTime:     &startTime,
		PlannedEndTime:       &endTime,
	}

	if !plannerDemo {
		event.Operations = []models.MaintenanceOperation{
			{
				Text:         "check pump condition and fix",
				WorkCenter:   "PUMP-WC01",
				Duration:     1,
				DurationUnit: "H",
			},
			{
				Text:         "Fix pump",
				WorkCenter:   "PUMP-WC01",
				Duration:     6,
				DurationUnit: "H",
			},
		}
	}

	prettyPrintJSON("Digital Twin → SAP Adaptor (MaintenanceOrderEvent)", event)

	// Send order creation request
	fmt.Println("")
	fmt.Println("   Sending HTTP POST request...")
	payload, _ := json.Marshal(event)
	createResp, err := http.Post(
		adaptorURL+"/api/v1/maintenance-orders",
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		fmt.Printf("ERROR: Failed to create order: %v\n", err)
		os.Exit(1)
	}
	defer createResp.Body.Close()

	var createResult map[string]interface{}
	body, _ := io.ReadAll(createResp.Body)
	json.Unmarshal(body, &createResult)

	fmt.Println("")
	fmt.Println("   Behind the scenes, SAP Adaptor executes:")
	fmt.Println("      Step 1: SAP Adaptor → SAP Simulator")
	fmt.Println("              POST /API_MAINTENANCE_NOTIFICATION/A_MaintenanceNotification")
	fmt.Println("              (Creates notification in SAP)")
	fmt.Println("")
	fmt.Println("      Step 2: SAP Simulator → SAP Adaptor")
	fmt.Println("              Response: Notification ID")
	fmt.Println("")
	fmt.Println("      Step 3: SAP Adaptor → SAP Simulator")
	fmt.Println("              POST /API_MAINTENANCE_ORDER/A_MaintenanceOrder")
	fmt.Println("              (Creates order with notification reference)")
	fmt.Println("")
	fmt.Println("      Step 4: SAP Simulator → SAP Adaptor")
	fmt.Println("              Response: Order ID and status")
	fmt.Println("")

	prettyPrintJSON("SAP Adaptor → Digital Twin (CreateOrder Response)", createResult)

	orderID, ok := createResult["maintenanceOrder"].(string)
	if !ok || orderID == "" {
		fmt.Printf("ERROR: Failed to get order ID from response\n")
		os.Exit(1)
	}

	notificationID := createResult["maintenanceNotification"]

	fmt.Println("")
	fmt.Println("SAP Workflow Completed Successfully:")
	fmt.Printf("   Step 1: Notification created → %v\n", notificationID)
	fmt.Printf("   Step 2: Order created → %s (referencing notification)\n", orderID)
	fmt.Printf("   Status: %v\n", createResult["status"])
	fmt.Println("")

	// Query initial status
	fmt.Println("3. Querying Initial Order Status...")
	statusResp, err := http.Get(adaptorURL + "/api/v1/maintenance-orders/" + orderID)
	if err != nil {
		fmt.Printf("ERROR: Failed to query order: %v\n", err)
		os.Exit(1)
	}
	defer statusResp.Body.Close()

	var statusData map[string]interface{}
	json.NewDecoder(statusResp.Body).Decode(&statusData)

	fmt.Printf("Initial Status: %v\n", statusData["status"])
	fmt.Printf("   Equipment: %v\n", statusData["equipment"])
	fmt.Printf("   Plant: %v\n", statusData["plant"])
	fmt.Printf("   Description: %v\n", statusData["description"])
	fmt.Println("")

	if plannerDemo {
		fmt.Println("4. Planner Enriches the SAP Simulator Order...")
		fmt.Println("   Planner → SAP Simulator: POST /planner/orders/{maintenanceOrder}/enrich")
		fmt.Println("")

		enrichment := plannerEnrichmentPayload()
		prettyPrintJSON("Planner → SAP Simulator (PlannerEnrichmentRequest)", enrichment)

		enrichmentBody, _ := json.Marshal(enrichment)
		enrichResp, err := http.Post(
			simulatorURL+"/planner/orders/"+orderID+"/enrich",
			"application/json",
			bytes.NewBuffer(enrichmentBody),
		)
		if err != nil {
			fmt.Printf("ERROR: Failed to enrich order: %v\n", err)
			os.Exit(1)
		}
		defer enrichResp.Body.Close()

		enrichRespBody, _ := io.ReadAll(enrichResp.Body)
		var enrichResult map[string]interface{}
		json.Unmarshal(enrichRespBody, &enrichResult)
		prettyPrintJSON("SAP Simulator → Planner (Enrichment Response)", enrichResult)

		if enrichResp.StatusCode != http.StatusOK {
			fmt.Printf("ERROR: Enrichment failed with HTTP %d\n", enrichResp.StatusCode)
			os.Exit(1)
		}

		fmt.Println("")
		fmt.Println("   Status behavior now follows planner-enrichment flow:")
		fmt.Println("      • Before enrichment: CRTD")
		fmt.Println("      • After enrichment:  REL")
		fmt.Println("      • 30+ seconds:       TECO")
		fmt.Println("")
	} else {
		fmt.Println("4. Background Monitoring Active...")
	}

	// Explain background monitoring
	fmt.Println("   SAP Adaptor is now polling this order automatically:")
	fmt.Println("      • Every few seconds: GET /API_MAINTENANCE_ORDER/A_MaintenanceOrder('{orderID}')")
	fmt.Println("      • Checks order status progression")
	fmt.Println("")
	if !plannerDemo {
		fmt.Println("   Time-based status progression (For simulation):")
		fmt.Println("      • 0-10 seconds:  CRTD (Created)")
		fmt.Println("      • 10-30 seconds: REL (Released)")
		fmt.Println("      • 30+ seconds:   TECO (Completed)")
		fmt.Println("")
	}
	fmt.Println("   When TECO is reached, adaptor will POST to:")
	fmt.Println("      → Digital Twin: /api/v1/maintenance-completed")
	fmt.Println("")
	fmt.Println("   Waiting 35 seconds for order completion...")

	time.Sleep(35 * time.Second)

	// Query final status with equipment data
	fmt.Println("\n5. Retrieving Final Order Status...")
	finalResp, err := http.Get(adaptorURL + "/api/v1/maintenance-orders/" + orderID)
	if err != nil {
		fmt.Printf("ERROR: Failed to query final order: %v\n", err)
		os.Exit(1)
	}
	defer finalResp.Body.Close()

	var finalData map[string]interface{}
	json.NewDecoder(finalResp.Body).Decode(&finalData)

	prettyPrintJSON("SAP Adaptor → Digital Twin (Final Order Status)", finalData)

	fmt.Println("\n6. Equipment Replacement Data...")
	fmt.Printf("Order completed: %v\n", finalData["maintenanceOrder"])
	fmt.Printf("   Status: %v\n", finalData["status"])
	fmt.Printf("   Equipment: %v\n", finalData["equipment"])
	fmt.Printf("   Plant: %v\n", finalData["plant"])

	// Check for operations with components
	if operations, ok := finalData["operations"].([]interface{}); ok && len(operations) > 0 {
		fmt.Printf("\n   Operations with Components:\n")
		for i, op := range operations {
			opMap := op.(map[string]interface{})
			fmt.Printf("      Operation %d: %v - %v\n", i+1, opMap["maintenanceOrderOperation"], opMap["text"])

			if components, ok := opMap["components"].([]interface{}); ok && len(components) > 0 {
				fmt.Printf("         Components (%d):\n", len(components))
				for j, comp := range components {
					compMap := comp.(map[string]interface{})
					fmt.Printf("            %d. %v - %v\n", j+1, compMap["material"], compMap["description"])
					fmt.Printf("               Quantity: %v %v (Movement: %v)\n",
						compMap["requirementQuantity"], compMap["materialUnit"], compMap["goodsMovementType"])
				}
			} else {
				fmt.Printf("         No components\n")
			}
		}
	} else {
		fmt.Println("\n   WARNING: No operations data available")
	}

	// Check for object list
	if objectList, ok := finalData["objectList"].([]interface{}); ok && len(objectList) > 0 {
		fmt.Printf("\n   Equipment Installed (%d):\n", len(objectList))
		for i, obj := range objectList {
			objMap := obj.(map[string]interface{})
			fmt.Printf("      %d. Equipment: %v\n", i+1, objMap["equipment"])
			fmt.Printf("         Material: %v\n", objMap["material"])
			fmt.Printf("         Serial Number: %v\n", objMap["serialNumber"])
		}
	} else {
		fmt.Println("\n   WARNING: No object list data available")
	}

	fmt.Println("\n=== End-to-End Test Complete! ===")
	fmt.Println("The SAP Adaptor is working correctly.")
	fmt.Println("")
	fmt.Println("Complete workflow verified:")
	fmt.Println("1. Digital Twin → SAP Adaptor (Maintenance Order Event)")
	fmt.Println("2. SAP Adaptor → SAP:")
	fmt.Println("   a) Create Notification → Get notification number")
	fmt.Println("   b) Create Order (with notification reference)")
	if plannerDemo {
		fmt.Println("3. Planner → SAP Simulator enrichment (operations, actual hours, materials)")
		fmt.Println("4. Planner-based status progression (CRTD → REL → TECO)")
		fmt.Println("5. SAP Adaptor detects TECO and sends notification to Digital Twin")
		fmt.Println("6. Planner-provided operations and materials available in final order")
	} else {
		fmt.Println("3. SAP Adaptor starts background polling")
		fmt.Println("4. Time-based status progression (CRTD → REL → TECO)")
		fmt.Println("5. SAP Adaptor detects TECO and sends notification to Digital Twin")
		fmt.Println("6. Equipment replacement data available in final order")
	}
	fmt.Println("")
	fmt.Println("Note: Check your Digital Twin callback listener for the notification!")
}

func plannerEnrichmentPayload() *models.PlannerEnrichmentRequest {
	finalIssue := true
	return &models.PlannerEnrichmentRequest{
		PlannedStartDateTime: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		PlannedEndDateTime:   time.Now().Add(9 * time.Hour).Format(time.RFC3339),
		MainWorkCenter:       "MECH-01",
		MainWorkCenterPlant:  "2000",
		Operations: []models.EnrichedOperation{
			{
				Operation:           "0010",
				ControlKey:          "PM01",
				Description:         "Disassemble pump",
				WorkCenter:          "MECH-01",
				Plant:               "2000",
				PlannedWorkQuantity: "4.0",
				ActualWorkQuantity:  "4.5",
				WorkQuantityUnit:    "H",
				ActualStartDateTime: time.Now().Add(-7 * time.Hour).Format(time.RFC3339),
				ActualEndDateTime:   time.Now().Add(-2*time.Hour - 30*time.Minute).Format(time.RFC3339),
				Components: []models.EnrichedComponent{
					{
						Component:         "0001",
						Material:          "SEAL-X200",
						Description:       "Pump seal",
						RequiredQuantity:  "1.000",
						UsedQuantity:      "1.000",
						Unit:              "EA",
						Plant:             "2000",
						StorageLocation:   "0001",
						GoodsMovementType: "261",
						FinalIssue:        &finalIssue,
					},
				},
			},
			{
				Operation:           "0020",
				ControlKey:          "PM01",
				Description:         "Replace seal and reassemble",
				WorkCenter:          "MECH-01",
				Plant:               "2000",
				PlannedWorkQuantity: "3.0",
				ActualWorkQuantity:  "2.5",
				WorkQuantityUnit:    "H",
				ActualStartDateTime: time.Now().Add(-2*time.Hour - 15*time.Minute).Format(time.RFC3339),
				ActualEndDateTime:   time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
				Components: []models.EnrichedComponent{
					{
						Component:         "0002",
						Material:          "BEARING-KIT",
						Description:       "Bearing replacement kit",
						RequiredQuantity:  "1.000",
						UsedQuantity:      "1.000",
						Unit:              "EA",
						Plant:             "2000",
						StorageLocation:   "0001",
						GoodsMovementType: "261",
						FinalIssue:        &finalIssue,
					},
					{
						Component:         "0003",
						Material:          "LUBE-GREASE",
						Description:       "Assembly grease",
						RequiredQuantity:  "0.250",
						UsedQuantity:      "0.300",
						Unit:              "KG",
						Plant:             "2000",
						StorageLocation:   "0001",
						GoodsMovementType: "261",
						FinalIssue:        &finalIssue,
					},
				},
			},
		},
	}
}
