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

	"censer-simulation/config"
	"censer-simulation/database"
	"censer-simulation/models"
	"censer-simulation/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Handler struct {
	dtuReceiver       *services.DtuReceiver
	gimbalSimulator   *services.GimbalSimulatorService
	sloshAnalyzer     *services.SloshAnalyzerService
	alarmWs           *services.AlarmWsService
	db                *database.DB
}

func NewHandlerWithServices(
	dtuReceiver *services.DtuReceiver,
	gimbalSimulator *services.GimbalSimulatorService,
	sloshAnalyzer *services.SloshAnalyzerService,
	alarmWs *services.AlarmWsService,
	db *database.DB,
) *Handler {
	return &Handler{
		dtuReceiver:     dtuReceiver,
		gimbalSimulator: gimbalSimulator,
		sloshAnalyzer:   sloshAnalyzer,
		alarmWs:         alarmWs,
		db:              db,
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

	if err := h.dtuReceiver.ValidateAndProcess(c, &req); err != nil {
		if err.Error() == "censer not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Data accepted and processing",
	})
}

func (h *Handler) GetMechanicalConfig(c *gin.Context) {
	c.JSON(http.StatusOK, config.Mechanical)
}

func (h *Handler) GetFluidConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"formulas":       config.Fluid.Formulas,
		"slosh_dynamics": config.Fluid.SloshDynamics,
		"default_formula": config.Fluid.DefaultFormula,
	})
}

func (h *Handler) GetMotionProfiles(c *gin.Context) {
	result := make(map[string]interface{})
	for key, profile := range config.Fluid.MotionProfiles {
		result[key] = profile
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetPerfumeFormulas(c *gin.Context) {
	c.JSON(http.StatusOK, config.Fluid.Formulas)
}

func (h *Handler) GetCensers(c *gin.Context) {
	ctx := context.Background()
	censers, err := h.db.GetCensers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, censers)
}

func (h *Handler) GetLatestSensorData(c *gin.Context) {
	ctx := context.Background()
	data, err := h.db.GetLatestSensorData(ctx)
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

	data, err := h.db.GetSensorDataByCenser(ctx, censerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetStabilityStats(c *gin.Context) {
	ctx := context.Background()
	stats, err := h.db.GetStabilityStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetActiveAlerts(c *gin.Context) {
	ctx := context.Background()
	alerts, err := h.db.GetActiveAlerts(ctx)
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

	alerts, err := h.db.GetAlertsByCenser(ctx, censerID, limit)
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

	if err := h.db.AcknowledgeAlert(ctx, alertID, req.AcknowledgedBy); err != nil {
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

	config, err := h.db.GetSimulationConfig(censerID)
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

	var result *models.SloshAnalysisResult

	if req.MotionType != nil {
		result, err = h.sloshAnalyzer.AnalyzeMotion(censerID.String(), *req.MotionType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if req.Frequency != nil && req.Amplitude != nil {
		duration := 10.0
		if req.Duration != nil {
			duration = *req.Duration
		}
		result, err = h.sloshAnalyzer.AnalyzeCustomMotion(censerID.String(), *req.Frequency, *req.Amplitude, duration)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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

	if err := h.db.InsertSloshAnalysis(ctx, analysisRecord); err != nil {
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

	history, err := h.db.GetSloshAnalysisByCenser(ctx, censerID, limit)
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

	freqs, amps, phases, err := h.sloshAnalyzer.FrequencyResponseAnalysis(censerID.String(), minFreq, maxFreq, numPoints)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	naturalInfo, err := h.sloshAnalyzer.GetNaturalFrequencyInfo(censerID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"frequencies":       freqs,
		"amplitudes":        amps,
		"phases":            phases,
		"natural_frequency": naturalInfo,
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
		Duration        float64               `json:"duration" binding:"required"`
		DT              float64               `json:"dt"`
		ExternalForce   *models.ExternalForce `json:"external_force"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	states, tilts, err := h.gimbalSimulator.SimulateResponse(censerID.String(), req.ExternalForce, req.Duration, req.DT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	h.alarmWs.HandleConnection(conn)
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"clients": h.alarmWs.ClientCount(),
		"time":    time.Now(),
		"services": gin.H{
			"dtu_receiver":      "running",
			"gimbal_simulator":  "running",
			"slosh_analyzer":    "running",
			"alarm_ws":          "running",
		},
	})
}
