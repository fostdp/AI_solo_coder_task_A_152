package services

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"censer-simulation/database"
	"censer-simulation/models"
)

type DtuReceiver struct {
	bus         *MessageBus
	db          *database.DB
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
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
}

func (r *DtuReceiver) Stop() {
	r.running = false
	r.cancel()
}

func (r *DtuReceiver) ValidateAndProcess(c *gin.Context, req *models.SensorDataRequest) error {
	if err := r.validateRequest(req); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	censer, err := r.db.GetCenserByCode(req.CenserCode)
	if err != nil {
		return fmt.Errorf("get censer: %w", err)
	}
	if censer == nil {
		return fmt.Errorf("censer not found: %s", req.CenserCode)
	}

	config, err := r.db.GetSimulationConfig(censer.ID)
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}
	if config == nil {
		return fmt.Errorf("simulation config not found for censer: %s", req.CenserCode)
	}

	force := &models.ExternalForce{
		AccelerationX: req.SloshAcceleration * 0.3,
		AccelerationY: req.SloshAcceleration * 0.4,
		AccelerationZ: req.SloshAcceleration * 0.5,
		Temperature:   req.Temperature,
	}

	innerVel := req.InnerRingVelocity
	outerVel := req.OuterRingVelocity
	bodyVel := req.BodyAngularVelocity
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
	temp := req.Temperature
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
		InnerRingAngle:      req.InnerRingAngle,
		OuterRingAngle:      req.OuterRingAngle,
		BodyTilt:            req.BodyTilt,
		SloshAcceleration:   req.SloshAcceleration,
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
		InnerRingAngle:      req.InnerRingAngle,
		OuterRingAngle:      req.OuterRingAngle,
		BodyTilt:            req.BodyTilt,
		SloshAcceleration:   req.SloshAcceleration,
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
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("message bus full, dropping sensor data")
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
