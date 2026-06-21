package services

import (
	"time"

	"github.com/google/uuid"

	"censer-simulation/models"
)

type MessageType string

const (
	MsgSensorRaw       MessageType = "sensor_raw"
	MsgBalanceResult   MessageType = "balance_result"
	MsgSloshResult     MessageType = "slosh_result"
	MsgAlert           MessageType = "alert"
	MsgBroadcastData   MessageType = "broadcast_data"
	MsgPersistData     MessageType = "persist_data"
)

type SensorRawMessage struct {
	Time                time.Time
	CenserID            uuid.UUID
	CenserCode          string
	CenserName          string
	InnerRingAngle      float64
	OuterRingAngle      float64
	BodyTilt            float64
	SloshAcceleration   float64
	InnerRingVelocity   *float64
	OuterRingVelocity   *float64
	BodyAngularVelocity *float64
	Temperature         *float64
	Force               *models.ExternalForce
	Config              *models.SimulationConfig
	SensorDataID        uuid.UUID
	SensorData          *models.SensorData
}

type BalanceResultMessage struct {
	Time          time.Time
	CenserID      uuid.UUID
	CenserCode    string
	CenserName    string
	BalanceScore  float64
	BodyTilt      float64
	InnerAngle    float64
	OuterAngle    float64
	InnerVelocity float64
	OuterVelocity float64
	BodyVelocity  float64
	SloshAccel    float64
	SensorData    *models.SensorData
}

type SloshResultMessage struct {
	Time           time.Time
	CenserID       uuid.UUID
	CenserCode     string
	SpillRisk      float64
	FluidDamping   float64
	SloshExcitation float64
	FillFactor     float64
	BalanceScore   float64
	BodyTilt       float64
	SensorData     *models.SensorData
}

type AlertMessage struct {
	ID             uuid.UUID
	CenserID       uuid.UUID
	CenserCode     string
	AlertType      string
	Severity       string
	Message        string
	ThresholdValue *float64
	ActualValue    *float64
	CreatedAt      time.Time
}

type BroadcastMessage struct {
	MessageType string
	Data        interface{}
	Time        time.Time
}

type PersistMessage struct {
	SensorData *models.SensorData
	AlertData  *models.Alert
}

type MessageBus struct {
	SensorRawCh       chan *SensorRawMessage
	BalanceResultCh   chan *BalanceResultMessage
	SloshResultCh     chan *SloshResultMessage
	AlertCh           chan *AlertMessage
	BroadcastCh       chan *BroadcastMessage
	PersistCh         chan *PersistMessage
}

func NewMessageBus(bufferSize int) *MessageBus {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	return &MessageBus{
		SensorRawCh:     make(chan *SensorRawMessage, bufferSize),
		BalanceResultCh: make(chan *BalanceResultMessage, bufferSize),
		SloshResultCh:   make(chan *SloshResultMessage, bufferSize),
		AlertCh:         make(chan *AlertMessage, bufferSize),
		BroadcastCh:     make(chan *BroadcastMessage, bufferSize),
		PersistCh:       make(chan *PersistMessage, bufferSize),
	}
}

func (mb *MessageBus) Close() {
	close(mb.SensorRawCh)
	close(mb.BalanceResultCh)
	close(mb.SloshResultCh)
	close(mb.AlertCh)
	close(mb.BroadcastCh)
	close(mb.PersistCh)
}
