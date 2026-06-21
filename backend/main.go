package main

import (
	"log"
	"net/http/pprof"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"censer-simulation/config"
	"censer-simulation/database"
	"censer-simulation/handlers"
	"censer-simulation/metrics"
	"censer-simulation/middleware"
	"censer-simulation/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	mechConfigPath := "config/mechanical_params.json"
	if p := os.Getenv("MECHANICAL_CONFIG"); p != "" {
		mechConfigPath = p
	}
	if _, err := config.LoadMechanicalConfig(mechConfigPath); err != nil {
		log.Fatalf("Failed to load mechanical config: %v", err)
	}
	log.Println("Mechanical config loaded successfully")

	fluidConfigPath := "config/fluid_params.json"
	if p := os.Getenv("FLUID_CONFIG"); p != "" {
		fluidConfigPath = p
	}
	if _, err := config.LoadFluidConfig(fluidConfigPath); err != nil {
		log.Fatalf("Failed to load fluid config: %v", err)
	}
	log.Println("Fluid config loaded successfully")

	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()
	db := database.GetDB()
	log.Println("Database initialized successfully")

	bus := services.NewMessageBus(256)
	defer bus.Close()
	log.Println("Message bus initialized")

	dtuReceiver := services.NewDtuReceiver(bus, db)

	mqttBroker := os.Getenv("MQTT_BROKER")
	mqttTopic := os.Getenv("MQTT_TOPIC")
	mqttClientID := os.Getenv("MQTT_CLIENT_ID")
	if mqttClientID == "" {
		mqttClientID = "censer-backend"
	}
	if mqttBroker != "" && mqttTopic != "" {
		dtuReceiver.EnableMQTT(mqttBroker, mqttTopic, mqttClientID)
		log.Printf("MQTT receiver enabled: broker=%s, topic=%s", mqttBroker, mqttTopic)
	}

	dtuReceiver.Start()
	log.Println("DTU Receiver started")

	gimbalSimulator := services.NewGimbalSimulatorService(bus)
	gimbalSimulator.Start()
	defer gimbalSimulator.Stop()
	log.Println("Gimbal Simulator Service started")

	sloshAnalyzer := services.NewSloshAnalyzerService(bus)
	sloshAnalyzer.Start()
	defer sloshAnalyzer.Stop()
	log.Println("Slosh Analyzer Service started")

	alarmWs := services.NewAlarmWsService(bus, db)
	alarmWs.Start()
	defer alarmWs.Stop()
	log.Println("Alarm & WebSocket Service started")

	go updateMetrics(alarmWs)

	h := handlers.NewHandlerWithServices(dtuReceiver, gimbalSimulator, sloshAnalyzer, alarmWs, db)

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.Gzip())
	r.Use(metrics.PrometheusMiddleware())

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

		api.GET("/config/mechanical", h.GetMechanicalConfig)
		api.GET("/config/fluid", h.GetFluidConfig)
		api.GET("/config/motion-profiles", h.GetMotionProfiles)
		api.GET("/config/formulas", h.GetPerfumeFormulas)

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

	r.GET("/metrics", metrics.PrometheusHandler())

	pprofGroup := r.Group("/debug/pprof")
	{
		pprofGroup.GET("/", gin.WrapF(pprof.Index))
		pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
		pprofGroup.POST("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
		pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
		pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
		pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}

	r.GET("/ws", h.WebSocketEndpoint)

	frontendPath := os.Getenv("FRONTEND_PATH")
	if frontendPath == "" {
		frontendPath = "../frontend"
	}
	r.Static("/static", frontendPath+"/static")
	r.StaticFile("/", frontendPath+"/index.html")
	r.StaticFile("/app.js", frontendPath+"/app.js")
	r.Static("/js", frontendPath+"/js")
	r.Static("/css", frontendPath+"/css")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func updateMetrics(alarmWs *services.AlarmWsService) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics.SetWebSocketClients(float64(alarmWs.ClientCount()))
	}
}
