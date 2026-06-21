package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"censer-simulation/alert"
	"censer-simulation/database"
	"censer-simulation/models"
	"censer-simulation/simulation"
	ws "censer-simulation/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Handler struct {
	hub          *ws.Hub
	alertManager *alert.AlertManager
}

func NewHandler(hub *ws.Hub, alertManager *alert.AlertManager) *Handler {
	return &Handler{
		hub:          hub,
		alertManager: alertManager,
	}
}

type SensorDataRequest struct {
	CenserCode          string   `json:"censer_code" binding:"required"`
	InnerRingAngle      float64  `json:"inner_ring_angle" binding:"required"`
	OuterRingAngle      float64  `json:"outer_ring_angle" binding:"required"`
	BodyTilt            float64  `json:"body_tilt" binding:"required"`
	SloshAcceleration   float64  `json:"slosh_acceleration" binding:"required"`
	InnerRingVelocity   *float64 `json:"inner_ring_velocity,omitempty"`
	OuterRingVelocity   *float64 `json:"outer_ring_velocity,omitempty"`
	BodyAngularVelocity *float64 `json:"body_angular_velocity,omitempty"`
	Temperature         *float64 `json:"temperature,omitempty"`
}

func (h *Handler) PostSensorData(c *gin.Context) {
	var req SensorDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	censer, err := database.GetCenserByCode(ctx, req.CenserCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Censer not found"})
		return
	}

	config, err := database.GetSimulationConfig(ctx, censer.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get simulation config"})
		return
	}

	force := &models.ExternalForce{
		AccelerationX: req.SloshAcceleration * 0.5,
		AccelerationY: req.SloshAcceleration * 0.3,
		AccelerationZ: req.SloshAcceleration * 0.2,
	}
	sim := simulation.NewGimbalSimulator(config)
	sim.State.InnerAngle = req.InnerRingAngle
	sim.State.OuterAngle = req.OuterRingAngle
	sim.State.BodyAngle = req.BodyTilt
	if req.InnerRingVelocity != nil {
		sim.State.InnerVelocity = *req.InnerRingVelocity
	}
	if req.OuterRingVelocity != nil {
		sim.State.OuterVelocity = *req.OuterRingVelocity
	}

	balanceScore := sim.CalculateBalanceScore()
	spillRisk := sim.CalculateSpillRisk()
	_ = force

	sensorData := &models.SensorData{
		Time:                time.Now(),
		CenserID:            censer.ID,
		InnerRingAngle:      req.InnerRingAngle,
		OuterRingAngle:      req.OuterRingAngle,
		BodyTilt:            req.BodyTilt,
		SloshAcceleration:   req.SloshAcceleration,
		InnerRingVelocity:   req.InnerRingVelocity,
		OuterRingVelocity:   req.OuterRingVelocity,
		BodyAngularVelocity: req.BodyAngularVelocity,
		Temperature:         req.Temperature,
		BalanceScore:        &balanceScore,
		SpillRisk:           &spillRisk,
	}

	if err := database.InsertSensorData(ctx, sensorData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert sensor data"})
		return
	}

	h.alertManager.CheckAndAlert(ctx, censer.ID, sensorData, config)

	h.hub.Broadcast("sensor_data", gin.H{
		"censer_id":            censer.ID,
		"censer_code":          censer.Code,
		"censer_name":          censer.Name,
		"time":                 sensorData.Time,
		"inner_ring_angle":     req.InnerRingAngle,
		"outer_ring_angle":     req.OuterRingAngle,
		"body_tilt":            req.BodyTilt,
		"slosh_acceleration":   req.SloshAcceleration,
		"balance_score":        balanceScore,
		"spill_risk":           spillRisk,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Data received successfully",
		"balance_score":  balanceScore,
		"spill_risk":     spillRisk,
	})
}

func (h *Handler) GetCensers(c *gin.Context) {
	ctx := context.Background()
	censers, err := database.GetCensers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, censers)
}

func (h *Handler) GetLatestSensorData(c *gin.Context) {
	ctx := context.Background()
	data, err := database.GetLatestSensorData(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetSensorDataByCenser(c *gin.Context) {
	ctx := context.Background()
	censerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid censer ID"})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	data, err := database.GetSensorDataByCenser(ctx, censerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetStabilityStats(c *gin.Context) {
	ctx := context.Background()
	stats, err := database.GetStabilityStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetActiveAlerts(c *gin.Context) {
	ctx := context.Background()
	alerts, err := database.GetActiveAlerts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

func (h *Handler) GetAlertsByCenser(c *gin.Context) {
	ctx := context.Background()
	censerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid censer ID"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	alerts, err := database.GetAlertsByCenser(ctx, censerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	ctx := context.Background()
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.AcknowledgedBy = "system"
	}

	if err := database.AcknowledgeAlert(ctx, alertID, req.AcknowledgedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged"})
}

func (h *Handler) GetSimulationConfig(c *gin.Context) {
	ctx := context.Background()
	censerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid censer ID"})
		return
	}

	config, err := database.GetSimulationConfig(ctx, censerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

type SloshAnalysisRequest struct {
	MotionType *string  `json:"motion_type,omitempty"`
	Frequency  *float64 `json:"frequency,omitempty"`
	Amplitude  *float64 `json:"amplitude,omitempty"`
	Duration   *float64 `json:"duration,omitempty"`
}

func (h *Handler) RunSloshAnalysis(c *gin.Context) {
	ctx := context.Background()
	censerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid censer ID"})
		return
	}

	var req SloshAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := database.GetSimulationConfig(ctx, censerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get simulation config"})
		return
	}

	analyzer := simulation.NewSloshAnalyzer(config)
	var result *models.SloshAnalysisResult

	if req.MotionType != nil {
		result = analyzer.AnalyzeMotion(*req.MotionType)
	} else if req.Frequency != nil && req.Amplitude != nil {
		duration := 10.0
		if req.Duration != nil {
			duration = *req.Duration
		}
		result = analyzer.AnalyzeCustomMotion(*req.Frequency, *req.Amplitude, duration)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either motion_type or both frequency and amplitude must be provided"})
		return
	}

	timeSeriesJSON, _ := json.Marshal(result.TimeSeries)
	timeSeriesStr := string(timeSeriesJSON)

	dampingRatio := result.DampingRatio
	resonanceFactor := result.ResonanceFactor
	maxTilt := result.MaxTiltAngle
	spillProb := result.SpillProbability
	balanceEff := result.BalanceEfficiency

	analysisRecord := &models.SloshAnalysis{
		CenserID:          censerID,
		AnalysisType:      "frequency_response",
		MotionType:        result.MotionType,
		Frequency:         result.Frequency,
		Amplitude:         result.Amplitude,
		DampingRatio:      &dampingRatio,
		ResonanceFactor:   &resonanceFactor,
		MaxTiltAngle:      &maxTilt,
		SpillProbability:  &spillProb,
		BalanceEfficiency: &balanceEff,
		AnalysisData:      &timeSeriesStr,
	}

	if err := database.InsertSloshAnalysis(ctx, analysisRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save analysis"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetSloshAnalysisHistory(c *gin.Context) {
	ctx := context.Background()
	censerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid censer ID"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	history, err := database.GetSloshAnalysisByCenser(ctx, censerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, history)
}

func (h *Handler) GetFrequencyResponse(c *gin.Context) {
	ctx := context.Background()
	censerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid censer ID"})
		return
	}

	config, err := database.GetSimulationConfig(ctx, censerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get simulation config"})
		return
	}

	analyzer := simulation.NewSloshAnalyzer(config)

	minFreq := 0.1
	maxFreq := 20.0
	numPoints := 100

	if mf := c.Query("min_freq"); mf != "" {
		if parsed, err := strconv.ParseFloat(mf, 64); err == nil {
			minFreq = parsed
		}
	}
	if mf := c.Query("max_freq"); mf != "" {
		if parsed, err := strconv.ParseFloat(mf, 64); err == nil {
			maxFreq = parsed
		}
	}
	if np := c.Query("points"); np != "" {
		if parsed, err := strconv.Atoi(np); err == nil {
			numPoints = parsed
		}
	}

	freqs, amps, phases := analyzer.FrequencyResponseAnalysis(minFreq, maxFreq, numPoints)
	naturalInfo := analyzer.GetNaturalFrequencyInfo()

	c.JSON(http.StatusOK, gin.H{
		"frequencies":        freqs,
		"amplitudes":         amps,
		"phases":             phases,
		"natural_frequency":  naturalInfo,
	})
}

func (h *Handler) RunGimbalSimulation(c *gin.Context) {
	ctx := context.Background()
	censerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid censer ID"})
		return
	}

	var req struct {
		Duration        float64            `json:"duration" binding:"required"`
		DT              float64            `json:"dt"`
		ExternalForce   *models.ExternalForce `json:"external_force"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := database.GetSimulationConfig(ctx, censerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get simulation config"})
		return
	}

	if req.DT <= 0 {
		req.DT = 0.01
	}
	if req.ExternalForce == nil {
		req.ExternalForce = &models.ExternalForce{
			AccelerationX: 0.5,
			AccelerationY: 0.3,
			AccelerationZ: 0.2,
		}
	}

	states, tilts := simulation.SimulateGimbalResponse(config, req.ExternalForce, req.Duration, req.DT)

	c.JSON(http.StatusOK, gin.H{
		"states": states,
		"tilts":  tilts,
	})
}

func (h *Handler) WebSocketEndpoint(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	client := ws.NewClient(h.hub, conn)
	h.hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"clients":      h.hub.ClientCount(),
		"time":         time.Now(),
	})
}
