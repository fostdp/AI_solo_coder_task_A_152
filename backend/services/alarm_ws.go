package services

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"censer-simulation/config"
	"censer-simulation/database"
	"censer-simulation/metrics"
	"censer-simulation/models"
)

type AlarmWsService struct {
	bus          *MessageBus
	db           *database.DB
	running      bool
	ctx          context.Context
	cancel       context.CancelFunc

	clients    map[*WsClient]bool
	register   chan *WsClient
	unregister chan *WsClient
	mu         sync.RWMutex

	recentAlerts  map[string]time.Time
	alertCooldown time.Duration
}

type WsClient struct {
	conn *websocket.Conn
	send chan []byte
}

func NewAlarmWsService(bus *MessageBus, db *database.DB) *AlarmWsService {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlarmWsService{
		bus:          bus,
		db:           db,
		ctx:          ctx,
		cancel:       cancel,
		clients:      make(map[*WsClient]bool),
		register:     make(chan *WsClient),
		unregister:   make(chan *WsClient),
		recentAlerts: make(map[string]time.Time),
		alertCooldown: 30 * time.Second,
	}
}

func (s *AlarmWsService) Start() {
	s.running = true
	go s.hubLoop()
	go s.alertLoop()
	go s.broadcastLoop()
	go s.persistenceLoop()
	go s.cleanupLoop()
}

func (s *AlarmWsService) Stop() {
	s.running = false
	s.cancel()
}

func (s *AlarmWsService) hubLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("WebSocket client connected. Total: %d", len(s.clients))
		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
			}
			s.mu.Unlock()
			log.Printf("WebSocket client disconnected. Total: %d", len(s.clients))
		}
	}
}

func (s *AlarmWsService) alertLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case sloshMsg, ok := <-s.bus.SloshResultCh:
			if !ok {
				return
			}
			s.processAlerts(sloshMsg)
		}
	}
}

func (s *AlarmWsService) broadcastLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case broadcastMsg, ok := <-s.bus.BroadcastCh:
			if !ok {
				return
			}
			s.broadcastToClients(broadcastMsg)
		}
	}
}

func (s *AlarmWsService) persistenceLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case persistMsg, ok := <-s.bus.PersistCh:
			if !ok {
				return
			}
			s.persistData(persistMsg)
		}
	}
}

func (s *AlarmWsService) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupOldAlerts()
		}
	}
}

func (s *AlarmWsService) processAlerts(sloshMsg *SloshResultMessage) {
	thresholds := s.getThresholds()

	tiltKey := "tilt_" + sloshMsg.CenserID.String()
	balanceKey := "balance_" + sloshMsg.CenserID.String()
	spillKey := "spill_" + sloshMsg.CenserID.String()

	if sloshMsg.BodyTilt > thresholds.TiltAlarmDeg && s.shouldAlert(tiltKey) {
		bodyTilt := sloshMsg.BodyTilt
		alert := s.createAlert(
			sloshMsg,
			"tilt_exceeded",
			"warning",
			"炉体倾角超过安全阈值",
			&thresholds.TiltAlarmDeg,
			&bodyTilt,
		)
		if sloshMsg.BodyTilt > thresholds.TiltCriticalDeg {
			alert.Severity = "critical"
			alert.Message = "炉体倾角严重超限，有洒香风险！"
		}
		s.dispatchAlert(alert)
	}

	if sloshMsg.BalanceScore < thresholds.BalanceAlarm && s.shouldAlert(balanceKey) {
		balanceScore := sloshMsg.BalanceScore
		alert := s.createAlert(
			sloshMsg,
			"balance_failure",
			"warning",
			"万向平衡机构稳定性下降",
			&thresholds.BalanceAlarm,
			&balanceScore,
		)
		if sloshMsg.BalanceScore < thresholds.BalanceCritical {
			alert.Severity = "critical"
			alert.Message = "万向平衡机构可能失效！"
		}
		s.dispatchAlert(alert)
	}

	if sloshMsg.SpillRisk > thresholds.SpillAlarm && s.shouldAlert(spillKey) {
		alert := s.createAlert(
			sloshMsg,
			"spill_risk",
			"warning",
			"检测到洒香风险",
			&thresholds.SpillAlarm,
			&sloshMsg.SpillRisk,
		)
		if sloshMsg.SpillRisk > thresholds.SpillCritical {
			alert.Severity = "critical"
			alert.Message = "高洒香概率警告！请立即采取措施"
		}
		s.dispatchAlert(alert)
	}
}

func (s *AlarmWsService) getThresholds() config.AlarmThresholds {
	if config.Mechanical != nil {
		return config.Mechanical.AlarmThresholds
	}
	return config.AlarmThresholds{
		TiltAlarmDeg:    15.0,
		TiltCriticalDeg: 22.5,
		BalanceAlarm:    0.3,
		BalanceCritical: 0.15,
		SpillAlarm:      0.5,
		SpillCritical:   0.65,
	}
}

func (s *AlarmWsService) shouldAlert(alertKey string) bool {
	s.mu.RLock()
	lastTime, exists := s.recentAlerts[alertKey]
	s.mu.RUnlock()

	if exists && time.Since(lastTime) < s.alertCooldown {
		return false
	}

	s.mu.Lock()
	s.recentAlerts[alertKey] = time.Now()
	s.mu.Unlock()

	return true
}

func (s *AlarmWsService) cleanupOldAlerts() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	for key, t := range s.recentAlerts {
		if t.Before(cutoff) {
			delete(s.recentAlerts, key)
		}
	}
}

func (s *AlarmWsService) createAlert(sloshMsg *SloshResultMessage, alertType, severity, message string, threshold, actual *float64) *AlertMessage {
	return &AlertMessage{
		ID:             uuid.New(),
		CenserID:       sloshMsg.CenserID,
		CenserCode:     sloshMsg.CenserCode,
		AlertType:      alertType,
		Severity:       severity,
		Message:        message,
		ThresholdValue: threshold,
		ActualValue:    actual,
		CreatedAt:      time.Now().UTC(),
	}
}

func (s *AlarmWsService) dispatchAlert(alert *AlertMessage) {
	metrics.RecordAlert(alert.AlertType, alert.Severity)
	dbAlert := &models.Alert{
		ID:             alert.ID,
		CenserID:       alert.CenserID,
		AlertType:      alert.AlertType,
		Severity:       alert.Severity,
		Message:        alert.Message,
		ThresholdValue: alert.ThresholdValue,
		ActualValue:    alert.ActualValue,
		Acknowledged:   false,
		CreatedAt:      alert.CreatedAt,
		UpdatedAt:      alert.CreatedAt,
	}

	select {
	case s.bus.PersistCh <- &PersistMessage{AlertData: dbAlert}:
	case <-time.After(50 * time.Millisecond):
		log.Printf("[alarm_ws] persist channel full, alert %s may not be saved", alert.ID)
	}

	select {
	case s.bus.AlertCh <- alert:
	case <-time.After(50 * time.Millisecond):
	}

	select {
	case s.bus.BroadcastCh <- &BroadcastMessage{
		MessageType: "alert",
		Data:        alert,
		Time:        alert.CreatedAt,
	}:
	case <-time.After(50 * time.Millisecond):
	}

	log.Printf("Alert triggered: [%s] %s - %s", alert.Severity, alert.AlertType, alert.Message)
}

func (s *AlarmWsService) broadcastToClients(msg *BroadcastMessage) {
	wsMsg := &models.WebsocketMessage{
		Type: msg.MessageType,
		Data: msg.Data,
		Time: msg.Time,
	}

	data, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for client := range s.clients {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(s.clients, client)
		}
	}
}

func (s *AlarmWsService) persistData(msg *PersistMessage) {
	if msg.SensorData != nil {
		if err := s.db.InsertSensorData(msg.SensorData); err != nil {
			log.Printf("Failed to persist sensor data: %v", err)
		}
	}
	if msg.AlertData != nil {
		if err := s.db.InsertAlert(context.Background(), msg.AlertData); err != nil {
			log.Printf("Failed to persist alert: %v", err)
		}
	}
}

func (s *AlarmWsService) HandleConnection(conn *websocket.Conn) {
	client := &WsClient{
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.register <- client

	defer func() {
		s.unregister <- client
		conn.Close()
	}()

	go s.writePump(client)
	s.readPump(client)
}

func (s *AlarmWsService) readPump(client *WsClient) {
	defer func() {
		s.unregister <- client
		client.conn.Close()
	}()

	client.conn.SetReadLimit(8192)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (s *AlarmWsService) writePump(client *WsClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *AlarmWsService) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}
