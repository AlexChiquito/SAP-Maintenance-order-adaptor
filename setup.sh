#!/bin/bash

echo "=== SAP Adaptor Setup (Simplified) ==="

# Copy environment template
if [ ! -f .env ]; then
    cp env.example .env
    echo "Created .env file from template"
    echo "Edit .env file with your configuration"
else
    echo ".env file already exists"
fi

# Create logs directory
mkdir -p logs
echo "Created logs directory"

# Build and start services
echo "Starting SAP Adaptor..."
docker-compose up --build -d

echo ""
echo "SAP Adaptor is running!"
echo "Maintenance Order API available at: http://localhost:8080"
echo "Health check: http://localhost:8080/health"
echo ""
echo "To view logs: docker-compose logs -f"
echo "To stop: docker-compose down"
echo ""
echo "API Documentation:"
echo "   - OpenAPI spec: http://localhost:8080/swagger/doc.json"
echo "   - Swagger UI: http://localhost:8080/swagger/index.html"
