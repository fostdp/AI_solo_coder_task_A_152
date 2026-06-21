package services

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"censer-simulation/database"
	"censer-simulation/metrics"
	"censer-simulation/models"
)

type DtuReceiver struct {
	bus         *MessageBus
	db          *database.DB
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	mqttEnabled bool
	mqttReceiver *MqttReceiver
}

func NewDtuReceiver(bus *MessageBus, db *database.DB) *DtuReceiver {
	ctx, cancel := context.WithCancel(context.Background())
	return &DtuReceiver{
		bus:    bus,
		db:     db,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (r *DtuReceiver) Start() {
	r.running = true
	if r.mqttEnabled && r.mqttReceiver != nil {
		go func() {
			if err := r.mqttReceiver.Start(); err != nil {
				fmt.Printf("[dtu] MQTT start failed: %v\n", err)
			}
		}()
	}
}

func (r *DtuReceiver) Stop() {
	r.running = false
	r.cancel()
	if r.mqttReceiver != nil {
		r.mqttReceiver.Stop()
	}
}

func (r *DtuReceiver) EnableMQTT(broker, topic, clientID string) {
	r.mqttEnabled = true
	r.mqttReceiver = NewMqttReceiver(r, broker, topic, clientID)
}

func (r *DtuReceiver) ValidateAndProcess(c *gin.Context, req *models.SensorDataRequest) error {
	if err := r.validateRequest(req); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	raw := &rawSensorInput{
		CenserCode:          req.CenserCode,
		InnerRingAngle:      req.InnerRingAngle,
		OuterRingAngle:      req.OuterRingAngle,
		BodyTilt:            req.BodyTilt,
		SloshAcceleration:   req.SloshAcceleration,
		InnerRingVelocity:   req.InnerRingVelocity,
		OuterRingVelocity:   req.OuterRingVelocity,
		BodyAngularVelocity: req.BodyAngularVelocity,
		Temperature:         req.Temperature,
	}

	censer, sensorDataID, err := r.processRaw(raw)
	if err != nil {
		return err
	}

	c.Set("censer", censer)
	c.Set("sensor_data_id", sensorDataID)
	return nil
}

func (r *DtuReceiver) validateRequest(req *models.SensorDataRequest) error {
	if req.CenserCode == "" {
		return fmt.Errorf("censer_code is required")
	}
	if len(req.CenserCode) > 50 {
		return fmt.Errorf("censer_code too long")
	}

	if req.InnerRingAngle < -90 || req.InnerRingAngle > 90 {
		return fmt.Errorf("inner_ring_angle must be between -90 and 90")
	}
	if req.OuterRingAngle < -90 || req.OuterRingAngle > 90 {
		return fmt.Errorf("outer_ring_angle must be between -90 and 90")
	}
	if req.BodyTilt < -180 || req.BodyTilt > 180 {
		return fmt.Errorf("body_tilt must be between -180 and 180")
	}

	if req.SloshAcceleration < 0 {
		return fmt.Errorf("slosh_acceleration must be non-negative")
	}
	if req.SloshAcceleration > 100 {
		return fmt.Errorf("slosh_acceleration too large")
	}

	if req.InnerRingVelocity != nil {
		if *req.InnerRingVelocity < -1000 || *req.InnerRingVelocity > 1000 {
			return fmt.Errorf("inner_ring_velocity out of range")
		}
	}
	if req.OuterRingVelocity != nil {
		if *req.OuterRingVelocity < -1000 || *req.OuterRingVelocity > 1000 {
			return fmt.Errorf("outer_ring_velocity out of range")
		}
	}
	if req.BodyAngularVelocity != nil {
		if *req.BodyAngularVelocity < -1000 || *req.BodyAngularVelocity > 1000 {
			return fmt.Errorf("body_angular_velocity out of range")
		}
	}
	if req.Temperature != nil {
		if *req.Temperature < -40 || *req.Temperature > 200 {
			return fmt.Errorf("temperature out of range")
		}
	}

	return nil
}

type rawSensorInput struct {
	CenserCode          string
	InnerRingAngle      float64
	OuterRingAngle      float64
	BodyTilt            float64
	SloshAcceleration   float64
	InnerRingVelocity   *float64
	OuterRingVelocity   *float64
	BodyAngularVelocity *float64
	Temperature         *float64
}

func (r *DtuReceiver) processRaw(raw *rawSensorInput) (*models.Censer, uuid.UUID, error) {
	censer, err := r.db.GetCenserByCode(raw.CenserCode)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("get censer: %w", err)
	}
	if censer == nil {
		return nil, uuid.Nil, fmt.Errorf("censer not found: %s", raw.CenserCode)
	}

	config, err := r.db.GetSimulationConfig(censer.ID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("get config: %w", err)
	}
	if config == nil {
		return nil, uuid.Nil, fmt.Errorf("simulation config not found for censer: %s", raw.CenserCode)
	}

	force := &models.ExternalForce{
		AccelerationX: raw.SloshAcceleration * 0.3,
		AccelerationY: raw.SloshAcceleration * 0.4,
		AccelerationZ: raw.SloshAcceleration * 0.5,
		Temperature:   raw.Temperature,
	}

	innerVel := raw.InnerRingVelocity
	outerVel := raw.OuterRingVelocity
	bodyVel := raw.BodyAngularVelocity
	if innerVel == nil {
		v := 0.0
		innerVel = &v
	}
	if outerVel == nil {
		v := 0.0
		outerVel = &v
	}
	if bodyVel == nil {
		v := 0.0
		bodyVel = &v
	}
	temp := raw.Temperature
	if temp == nil {
		t := 25.0
		temp = &t
	}

	sensorDataID := uuid.New()
	now := time.Now().UTC()

	sensorData := &models.SensorData{
		ID:                  sensorDataID,
		CenserID:            censer.ID,
		Timestamp:           now,
		InnerRingAngle:      raw.InnerRingAngle,
		OuterRingAngle:      raw.OuterRingAngle,
		BodyTilt:            raw.BodyTilt,
		SloshAcceleration:   raw.SloshAcceleration,
		InnerRingVelocity:   *innerVel,
		OuterRingVelocity:   *outerVel,
		BodyAngularVelocity: *bodyVel,
		Temperature:         *temp,
		CreatedAt:           now,
	}

	rawMsg := &SensorRawMessage{
		Time:                now,
		CenserID:            censer.ID,
		CenserCode:          censer.Code,
		CenserName:          censer.Name,
		InnerRingAngle:      raw.InnerRingAngle,
		OuterRingAngle:      raw.OuterRingAngle,
		BodyTilt:            raw.BodyTilt,
		SloshAcceleration:   raw.SloshAcceleration,
		InnerRingVelocity:   innerVel,
		OuterRingVelocity:   outerVel,
		BodyAngularVelocity: bodyVel,
		Temperature:         temp,
		Force:               force,
		Config:              config,
		SensorDataID:        sensorDataID,
		SensorData:          sensorData,
	}

	select {
	case r.bus.SensorRawCh <- rawMsg:
	default:
		return nil, uuid.Nil, fmt.Errorf("message bus full, dropping sensor data")
	}

	metrics.RecordSensorData(censer.Code)

	return censer, sensorDataID, nil
}
