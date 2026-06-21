package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "censer_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "censer_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	sensorDataReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "censer_sensor_data_received_total",
			Help: "Total number of sensor data points received",
		},
		[]string{"censer_code"},
	)

	alertsTriggered = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "censer_alerts_triggered_total",
			Help: "Total number of alerts triggered",
		},
		[]string{"alert_type", "severity"},
	)

	balanceScoreGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "censer_balance_score",
			Help: "Current balance score",
		},
		[]string{"censer_code"},
	)

	spillRiskGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "censer_spill_risk",
			Help: "Current spill risk",
		},
		[]string{"censer_code"},
	)

	tiltAngleGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "censer_body_tilt_degrees",
			Help: "Current body tilt angle in degrees",
		},
		[]string{"censer_code"},
	)

	wsClientsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "censer_websocket_clients",
			Help: "Number of connected WebSocket clients",
		},
	)

	mqttMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "censer_mqtt_messages_received_total",
			Help: "Total number of MQTT messages received",
		},
		[]string{"topic"},
	)
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func RecordSensorData(censerCode string) {
	sensorDataReceived.WithLabelValues(censerCode).Inc()
}

func RecordAlert(alertType, severity string) {
	alertsTriggered.WithLabelValues(alertType, severity).Inc()
}

func SetBalanceScore(censerCode string, score float64) {
	balanceScoreGauge.WithLabelValues(censerCode).Set(score)
}

func SetSpillRisk(censerCode string, risk float64) {
	spillRiskGauge.WithLabelValues(censerCode).Set(risk)
}

func SetTiltAngle(censerCode string, tilt float64) {
	tiltAngleGauge.WithLabelValues(censerCode).Set(tilt)
}

func SetWebSocketClients(count float64) {
	wsClientsGauge.Set(count)
}

func RecordMQTTMessage(topic string) {
	mqttMessagesReceived.WithLabelValues(topic).Inc()
}
