package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"censer-simulation/metrics"
)

type MqttReceiver struct {
	dtu      *DtuReceiver
	client   mqtt.Client
	broker   string
	topic    string
	clientID string
	running  bool
}

func NewMqttReceiver(dtu *DtuReceiver, broker, topic, clientID string) *MqttReceiver {
	return &MqttReceiver{
		dtu:      dtu,
		broker:   broker,
		topic:    topic,
		clientID: clientID,
	}
}

func (m *MqttReceiver) Start() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(m.broker)
	opts.SetClientID(m.clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)

	opts.OnConnect = func(client mqtt.Client) {
		log.Printf("[mqtt] Connected to broker: %s", m.broker)
		if token := client.Subscribe(m.topic, 1, m.handleMessage); token.Wait() && token.Error() != nil {
			log.Printf("[mqtt] Subscribe error: %v", token.Error())
		} else {
			log.Printf("[mqtt] Subscribed to topic: %s", m.topic)
		}
	}

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Printf("[mqtt] Connection lost: %v", err)
	}

	m.client = mqtt.NewClient(opts)

	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt connect failed: %w", token.Error())
	}

	m.running = true
	return nil
}

func (m *MqttReceiver) Stop() {
	if m.client != nil && m.client.IsConnected() {
		m.client.Unsubscribe(m.topic)
		m.client.Disconnect(250)
	}
	m.running = false
}

func (m *MqttReceiver) handleMessage(client mqtt.Client, msg mqtt.Message) {
	metrics.RecordMQTTMessage(msg.Topic())

	var sensorMsg MqttSensorMessage
	if err := json.Unmarshal(msg.Payload(), &sensorMsg); err != nil {
		log.Printf("[mqtt] Failed to parse message: %v", err)
		return
	}

	if err := validateMqttMessage(&sensorMsg); err != nil {
		log.Printf("[mqtt] Validation failed: %v", err)
		return
	}

	raw := &rawSensorInput{
		CenserCode:          sensorMsg.CenserCode,
		InnerRingAngle:      sensorMsg.InnerRingAngle,
		OuterRingAngle:      sensorMsg.OuterRingAngle,
		BodyTilt:            sensorMsg.BodyTilt,
		SloshAcceleration:   sensorMsg.SloshAcceleration,
		InnerRingVelocity:   sensorMsg.InnerRingVelocity,
		OuterRingVelocity:   sensorMsg.OuterRingVelocity,
		BodyAngularVelocity: sensorMsg.BodyAngularVelocity,
		Temperature:         sensorMsg.Temperature,
	}

	_, _, err := m.dtu.processRaw(raw)
	if err != nil {
		log.Printf("[mqtt] Process sensor data failed: %v", err)
		return
	}
}

type MqttSensorMessage struct {
	CenserCode          string   `json:"censer_code" binding:"required"`
	InnerRingAngle      float64  `json:"inner_ring_angle" binding:"required"`
	OuterRingAngle      float64  `json:"outer_ring_angle" binding:"required"`
	BodyTilt            float64  `json:"body_tilt" binding:"required"`
	SloshAcceleration   float64  `json:"slosh_acceleration" binding:"required"`
	InnerRingVelocity   *float64 `json:"inner_ring_velocity,omitempty"`
	OuterRingVelocity   *float64 `json:"outer_ring_velocity,omitempty"`
	BodyAngularVelocity *float64 `json:"body_angular_velocity,omitempty"`
	Temperature         *float64 `json:"temperature,omitempty"`
	Timestamp           *int64   `json:"timestamp,omitempty"`
}

func validateMqttMessage(msg *MqttSensorMessage) error {
	if msg.CenserCode == "" {
		return fmt.Errorf("censer_code is required")
	}
	if len(msg.CenserCode) > 50 {
		return fmt.Errorf("censer_code too long")
	}

	if msg.InnerRingAngle < -90 || msg.InnerRingAngle > 90 {
		return fmt.Errorf("inner_ring_angle must be between -90 and 90")
	}
	if msg.OuterRingAngle < -90 || msg.OuterRingAngle > 90 {
		return fmt.Errorf("outer_ring_angle must be between -90 and 90")
	}
	if msg.BodyTilt < -180 || msg.BodyTilt > 180 {
		return fmt.Errorf("body_tilt must be between -180 and 180")
	}

	if msg.SloshAcceleration < 0 {
		return fmt.Errorf("slosh_acceleration must be non-negative")
	}
	if msg.SloshAcceleration > 100 {
		return fmt.Errorf("slosh_acceleration too large")
	}

	if msg.InnerRingVelocity != nil {
		if *msg.InnerRingVelocity < -1000 || *msg.InnerRingVelocity > 1000 {
			return fmt.Errorf("inner_ring_velocity out of range")
		}
	}
	if msg.OuterRingVelocity != nil {
		if *msg.OuterRingVelocity < -1000 || *msg.OuterRingVelocity > 1000 {
			return fmt.Errorf("outer_ring_velocity out of range")
		}
	}
	if msg.BodyAngularVelocity != nil {
		if *msg.BodyAngularVelocity < -1000 || *msg.BodyAngularVelocity > 1000 {
			return fmt.Errorf("body_angular_velocity out of range")
		}
	}
	if msg.Temperature != nil {
		if *msg.Temperature < -40 || *msg.Temperature > 200 {
			return fmt.Errorf("temperature out of range")
		}
	}

	return nil
}

func (m *MqttReceiver) IsConnected() bool {
	if m.client == nil {
		return false
	}
	return m.client.IsConnected()
}
