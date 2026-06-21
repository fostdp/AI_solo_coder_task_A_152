package alert

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"censer-simulation/database"
	"censer-simulation/models"
	ws "censer-simulation/websocket"
)

type AlertManager struct {
	hub            *ws.Hub
	mu             sync.RWMutex
	recentAlerts   map[string]time.Time
	alertCooldown  time.Duration
}

func NewAlertManager(hub *ws.Hub) *AlertManager {
	return &AlertManager{
		hub:           hub,
		recentAlerts:  make(map[string]time.Time),
		alertCooldown: 30 * time.Second,
	}
}

func (am *AlertManager) shouldAlert(alertKey string) bool {
	am.mu.RLock()
	lastTime, exists := am.recentAlerts[alertKey]
	am.mu.RUnlock()

	if exists && time.Since(lastTime) < am.alertCooldown {
		return false
	}

	am.mu.Lock()
	am.recentAlerts[alertKey] = time.Now()
	am.mu.Unlock()

	return true
}

func (am *AlertManager) CheckAndAlert(ctx context.Context, censerID uuid.UUID, sensorData *models.SensorData, config *models.SimulationConfig) []*models.Alert {
	var alerts []*models.Alert

	tiltKey := "tilt_" + censerID.String()
	if sensorData.BodyTilt > config.TiltAlarmThreshold && am.shouldAlert(tiltKey) {
		alert := &models.Alert{
			CenserID:       censerID,
			AlertType:      "tilt_exceeded",
			Severity:       "warning",
			Message:        "炉体倾角超过安全阈值",
			ThresholdValue: &config.TiltAlarmThreshold,
			ActualValue:    &sensorData.BodyTilt,
			SensorDataTime: &sensorData.Time,
		}
		if sensorData.BodyTilt > config.TiltAlarmThreshold*1.5 {
			alert.Severity = "critical"
			alert.Message = "炉体倾角严重超限，有洒香风险！"
		}
		alerts = append(alerts, alert)
	}

	balanceKey := "balance_" + censerID.String()
	if sensorData.BalanceScore != nil && *sensorData.BalanceScore < config.BalanceAlarmThreshold && am.shouldAlert(balanceKey) {
		balanceAlert := &models.Alert{
			CenserID:       censerID,
			AlertType:      "balance_failure",
			Severity:       "warning",
			Message:        "万向平衡机构稳定性下降",
			ThresholdValue: &config.BalanceAlarmThreshold,
			ActualValue:    sensorData.BalanceScore,
			SensorDataTime: &sensorData.Time,
		}
		if *sensorData.BalanceScore < config.BalanceAlarmThreshold*0.5 {
			balanceAlert.Severity = "critical"
			balanceAlert.Message = "万向平衡机构可能失效！"
		}
		alerts = append(alerts, balanceAlert)
	}

	spillKey := "spill_" + censerID.String()
	if sensorData.SpillRisk != nil && *sensorData.SpillRisk > config.SpillAlarmThreshold && am.shouldAlert(spillKey) {
		spillAlert := &models.Alert{
			CenserID:       censerID,
			AlertType:      "spill_risk",
			Severity:       "warning",
			Message:        "检测到洒香风险",
			ThresholdValue: &config.SpillAlarmThreshold,
			ActualValue:    sensorData.SpillRisk,
			SensorDataTime: &sensorData.Time,
		}
		if *sensorData.SpillRisk > config.SpillAlarmThreshold*1.3 {
			spillAlert.Severity = "critical"
			spillAlert.Message = "高洒香概率警告！请立即采取措施"
		}
		alerts = append(alerts, spillAlert)
	}

	for _, alert := range alerts {
		if err := database.InsertAlert(ctx, alert); err != nil {
			log.Printf("Failed to insert alert: %v", err)
			continue
		}
		am.hub.Broadcast("alert", alert)
		log.Printf("Alert triggered: [%s] %s - %s", alert.Severity, alert.AlertType, alert.Message)
	}

	return alerts
}

func (am *AlertManager) CleanupOldAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	for key, t := range am.recentAlerts {
		if t.Before(cutoff) {
			delete(am.recentAlerts, key)
		}
	}
}

func (am *AlertManager) StartCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			am.CleanupOldAlerts()
		}
	}()
}
