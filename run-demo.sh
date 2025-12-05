#!/bin/bash

# SAP Adaptor Demo Runner
# This script starts the simulator and runs the data flow demo

set -e

echo "🚀 Starting SAP Adaptor Demo..."
echo

# Check if binaries exist
if [ ! -f "bin/simulator" ] || [ ! -f "bin/demo-flow" ]; then
    echo "📦 Building binaries..."
    go build -o bin/simulator ./cmd/simulator
    go build -o bin/demo-flow ./cmd/demo-flow
    echo "✅ Build complete"
    echo
fi

# Start simulator in background
echo "🔧 Starting SAP Simulator on port 8081..."
./bin/simulator > /tmp/sap-simulator.log 2>&1 &
SIMULATOR_PID=$!
echo "   PID: $SIMULATOR_PID"

# Wait for simulator to be ready
echo "⏳ Waiting for simulator to be ready..."
for i in {1..10}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo "✅ Simulator is ready!"
        echo
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Simulator failed to start"
        kill $SIMULATOR_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# Run the demo
echo "🎬 Running Data Flow Demo..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
./bin/demo-flow 2>&1 | grep -v "^time="

# Cleanup
echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧹 Cleaning up..."
kill $SIMULATOR_PID 2>/dev/null || true
echo "✅ Simulator stopped"
echo
echo "📝 Simulator logs available at: /tmp/sap-simulator.log"
echo "💡 To run manually:"
echo "   Terminal 1: ./bin/simulator"
echo "   Terminal 2: ./bin/demo-flow"
