#!/bin/bash

# SAP Adaptor API Test Script
# This script demonstrates how to use the SAP Adaptor API
BASE_URL="http://localhost:8080"
API_BASE_URL="$BASE_URL/api/v1"

echo "=== SAP Adaptor API Test Script ==="
echo

# Test health endpoint
echo "1. Testing health endpoint..."
curl -s "$BASE_URL/health" | jq .
echo

# Test creating a maintenance order
echo "3. Creating a maintenance order..."
ORDER_RESPONSE=$(curl -s -X POST "$API_BASE_URL/maintenance-orders" \
  -H "Content-Type: application/json" \
  -d '{
    "equipmentId": "10000045",
    "functionalLocation": "FL100-200-300",
    "plant": "1000",
    "description": "Replace pump seal due to leakage",
    "priority": "3",
    "maintenanceOrderType": "PM01",
    "plannedStartTime": "2025-08-21T08:00:00Z",
    "plannedEndTime": "2025-08-21T16:00:00Z",
    "operations": [
      {
        "text": "Disassemble pump",
        "workCenter": "PUMP-WC01",
        "duration": 4,
        "durationUnit": "H"
      },
      {
        "text": "Replace seal and test",
        "workCenter": "PUMP-WC01",
        "duration": 2,
        "durationUnit": "H"
      }
    ]
  }')

echo "$ORDER_RESPONSE" | jq .

# Extract order ID from response
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.orderId')

if [ "$ORDER_ID" != "null" ] && [ "$ORDER_ID" != "" ]; then
    echo
    echo "4. Getting order status for order ID: $ORDER_ID"
    curl -s "$API_BASE_URL/maintenance-orders/$ORDER_ID" | jq .
    echo
    
    echo "5. Testing maintenance done event..."
    curl -s -X POST "$API_BASE_URL/maintenance-done" \
      -H "Content-Type: application/json" \
      -d "{
        \"orderId\": \"$ORDER_ID\",
        \"status\": \"TECO\",
        \"completedAt\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
        \"actualWorkHours\": 6.0,
        \"notes\": \"Maintenance completed successfully\"
      }" | jq .
else
    echo "Failed to create order or extract order ID"
fi

echo
echo "=== Test completed ==="


