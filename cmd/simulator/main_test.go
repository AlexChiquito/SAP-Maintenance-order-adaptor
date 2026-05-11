package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sap-adaptor/internal/models"
	"sap-adaptor/internal/sap"

	"github.com/sirupsen/logrus"
)

func TestPlannerInputModeRequired(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		envValue  string
		want      bool
	}{
		{name: "FlagRequired", flagValue: "required", envValue: "", want: true},
		{name: "EnvRequired", flagValue: "", envValue: "required", want: true},
		{name: "FlagWinsDisabled", flagValue: "disabled", envValue: "required", want: false},
		{name: "Unset", flagValue: "", envValue: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plannerInputModeRequired(tt.flagValue, tt.envValue); got != tt.want {
				t.Fatalf("Expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestHandlePlannerOrder(t *testing.T) {
	t.Run("PlannerInputDisabled", func(t *testing.T) {
		setupPlannerHandlerTest(false)

		orderID := createStoredTestOrder()
		rr := postPlannerEnrichment(orderID, testPlannerEnrichmentPayload(t))

		if rr.Code != http.StatusConflict {
			t.Fatalf("Expected 409, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		setupPlannerHandlerTest(true)

		rr := postPlannerEnrichment("400000404", testPlannerEnrichmentPayload(t))

		if rr.Code != http.StatusNotFound {
			t.Fatalf("Expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		setupPlannerHandlerTest(true)

		orderID := createStoredTestOrder()
		req := httptest.NewRequest(http.MethodPost, "/planner/orders/"+orderID+"/enrich", bytes.NewBufferString("{"))
		rr := httptest.NewRecorder()

		handlePlannerOrder(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("IncompleteRequest", func(t *testing.T) {
		setupPlannerHandlerTest(true)

		orderID := createStoredTestOrder()
		payload, err := json.Marshal(models.PlannerEnrichmentRequest{
			Operations: []models.EnrichedOperation{{Description: "Missing required fields"}},
		})
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}
		rr := postPlannerEnrichment(orderID, payload)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("AlreadyEnriched", func(t *testing.T) {
		setupPlannerHandlerTest(true)

		orderID := createStoredTestOrder()
		first := postPlannerEnrichment(orderID, testPlannerEnrichmentPayload(t))
		if first.Code != http.StatusOK {
			t.Fatalf("Expected first enrichment 200, got %d: %s", first.Code, first.Body.String())
		}

		second := postPlannerEnrichment(orderID, testPlannerEnrichmentPayload(t))
		if second.Code != http.StatusConflict {
			t.Fatalf("Expected repeated enrichment 409, got %d: %s", second.Code, second.Body.String())
		}
	})

	t.Run("Success", func(t *testing.T) {
		setupPlannerHandlerTest(true)

		orderID := createStoredTestOrder()
		rr := postPlannerEnrichment(orderID, testPlannerEnrichmentPayload(t))

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp models.PlannerEnrichmentResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if resp.MaintenanceOrder != orderID {
			t.Errorf("Expected order %s, got %s", orderID, resp.MaintenanceOrder)
		}
		if resp.Status != "REL" {
			t.Errorf("Expected status REL, got %s", resp.Status)
		}
	})
}

func setupPlannerHandlerTest(plannerInputRequired bool) {
	logger = logrus.New()
	logger.SetOutput(bytes.NewBuffer(nil))
	mockGenerator = sap.NewMockGenerator()
	mockGenerator.SetPlannerInputRequired(plannerInputRequired)
}

func createStoredTestOrder() string {
	resp := mockGenerator.CreateMockOrderResponse(&models.SAPOrderRequest{
		MaintenanceOrderType:    "PM01",
		Description:             "Planner order",
		Equipment:               "PUMP-100",
		FunctionalLocation:      "TEST-LOC",
		Plant:                   "1000",
		MaintenanceNotification: "200000123",
	})
	return resp.D.MaintenanceOrder
}

func postPlannerEnrichment(orderID string, payload []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/planner/orders/"+orderID+"/enrich", bytes.NewReader(payload))
	rr := httptest.NewRecorder()

	handlePlannerOrder(rr, req)

	return rr
}

func testPlannerEnrichmentPayload(t *testing.T) []byte {
	t.Helper()

	payload, err := json.Marshal(models.PlannerEnrichmentRequest{
		Operations: []models.EnrichedOperation{
			{
				Description:         "Disassemble pump",
				WorkCenter:          "MECH-01",
				Plant:               "1000",
				PlannedWorkQuantity: "4.0",
				WorkQuantityUnit:    "H",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	return payload
}
