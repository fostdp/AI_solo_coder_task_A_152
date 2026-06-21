package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"censer-simulation/alert"
	"censer-simulation/database"
	"censer-simulation/handlers"
	ws "censer-simulation/websocket"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()
	log.Println("Database initialized successfully")

	hub := ws.NewHub()
	go hub.Run()
	log.Println("WebSocket hub started")

	alertManager := alert.NewAlertManager(hub)
	alertManager.StartCleanupRoutine()
	log.Println("Alert manager started")

	h := handlers.NewHandler(hub, alertManager)

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api/v1")
	{
		api.GET("/health", h.HealthCheck)

		api.GET("/censers", h.GetCensers)
		api.GET("/censers/:id/config", h.GetSimulationConfig)

		api.POST("/sensor-data", h.PostSensorData)
		api.GET("/sensor-data/latest", h.GetLatestSensorData)
		api.GET("/censers/:id/sensor-data", h.GetSensorDataByCenser)

		api.GET("/stability-stats", h.GetStabilityStats)

		api.GET("/alerts/active", h.GetActiveAlerts)
		api.GET("/censers/:id/alerts", h.GetAlertsByCenser)
		api.POST("/alerts/:id/acknowledge", h.AcknowledgeAlert)

		api.POST("/censers/:id/slosh-analysis", h.RunSloshAnalysis)
		api.GET("/censers/:id/slosh-analysis", h.GetSloshAnalysisHistory)
		api.GET("/censers/:id/frequency-response", h.GetFrequencyResponse)

		api.POST("/censers/:id/gimbal-simulation", h.RunGimbalSimulation)
	}

	r.GET("/ws", h.WebSocketEndpoint)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
