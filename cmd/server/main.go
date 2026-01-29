package main

import (
	"log"
	"sap-adaptor/internal/config"
	"sap-adaptor/internal/handlers"
	"sap-adaptor/internal/sap"
	"sap-adaptor/internal/services"

	_ "sap-adaptor/docs" // Swagger API documentation

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title SAP Adaptor API
// @version 1.0
// @description SAP Adaptor for Maintenance Order Event processing
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Log configuration
	logger.WithFields(logrus.Fields{
		"sapBaseURL":     cfg.SAP.BaseURL,
		"digitalTwinURL": cfg.DigitalTwin.BaseURL,
	}).Info("Configuration loaded")

	// Initialize SAP client
	sapClient := sap.NewClient(cfg.SAP, logger)

	// Initialize services
	maintenanceService := services.NewMaintenanceService(sapClient, logger)

	// Configure Digital Twin integration
	if cfg.DigitalTwin.BaseURL != "" {
		maintenanceService.SetDigitalTwinConfig(cfg.DigitalTwin.BaseURL, cfg.DigitalTwin.APIKey)
		logger.WithField("url", cfg.DigitalTwin.BaseURL).Info("Digital Twin integration configured")
	} else {
		logger.Warn("Digital Twin URL not configured - completion notifications will not be sent")
	}

	// Initialize handlers
	maintenanceHandler := handlers.NewMaintenanceHandler(maintenanceService, logger)

	// Setup router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Setup routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/maintenance-orders", maintenanceHandler.CreateMaintenanceOrder)
		v1.GET("/maintenance-orders/:id", maintenanceHandler.GetMaintenanceOrder)
		v1.POST("/maintenance-done", maintenanceHandler.HandleMaintenanceDone)
	}

	// System routes
	router.GET("/health", maintenanceHandler.HealthCheck)

	// Swagger documentation routes
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start server
	address := cfg.Server.Host + ":" + cfg.Server.Port
	logger.Infof("Starting SAP Adaptor server on %s", address)
	log.Fatal(router.Run(address))
}
