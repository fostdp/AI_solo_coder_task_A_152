package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"censer-simulation/config"
	"censer-simulation/metrics"
	"censer-simulation/models"
	"censer-simulation/simulation"
)

type GimbalSimulatorService struct {
	bus         *MessageBus
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	simulators  map[string]*simulation.GimbalSimulator
}

func NewGimbalSimulatorService(bus *MessageBus) *GimbalSimulatorService {
	ctx, cancel := context.WithCancel(context.Background())
	return &GimbalSimulatorService{
		bus:        bus,
		ctx:        ctx,
		cancel:     cancel,
		simulators: make(map[string]*simulation.GimbalSimulator),
	}
}

func (s *GimbalSimulatorService) Start() {
	s.running = true
	go s.processLoop()
}

func (s *GimbalSimulatorService) Stop() {
	s.running = false
	s.cancel()
}

func (s *GimbalSimulatorService) processLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case rawMsg, ok := <-s.bus.SensorRawCh:
			if !ok {
				return
			}
			s.processSensorData(rawMsg)
		}
	}
}

func (s *GimbalSimulatorService) processSensorData(msg *SensorRawMessage) {
	sim := s.getOrCreateSimulator(msg)

	sim.State.InnerAngle = msg.InnerRingAngle
	sim.State.OuterAngle = msg.OuterRingAngle
	sim.State.BodyAngle = msg.BodyTilt

	innerVel := 0.0
	if msg.InnerRingVelocity != nil {
		innerVel = *msg.InnerRingVelocity
	}
	outerVel := 0.0
	if msg.OuterRingVelocity != nil {
		outerVel = *msg.OuterRingVelocity
	}
	bodyVel := 0.0
	if msg.BodyAngularVelocity != nil {
		bodyVel = *msg.BodyAngularVelocity
	}

	sim.State.InnerVelocity = innerVel
	sim.State.OuterVelocity = outerVel
	sim.State.BodyVelocity = bodyVel

	if msg.Force != nil {
		dt := 0.016
		sim.Step(dt, msg.Force)
	}

	bodyTilt := sim.CalculateBodyTilt()
	balanceScore := sim.CalculateBalanceScore()

	metrics.SetBalanceScore(msg.CenserCode, balanceScore)
	metrics.SetTiltAngle(msg.CenserCode, bodyTilt)

	balanceMsg := &BalanceResultMessage{
		Time:          msg.Time,
		CenserID:      msg.CenserID,
		CenserCode:    msg.CenserCode,
		CenserName:    msg.CenserName,
		BalanceScore:  balanceScore,
		BodyTilt:      bodyTilt,
		InnerAngle:    msg.InnerRingAngle,
		OuterAngle:    msg.OuterRingAngle,
		InnerVelocity: innerVel,
		OuterVelocity: outerVel,
		BodyVelocity:  bodyVel,
		SloshAccel:    msg.SloshAcceleration,
		SensorData:    msg.SensorData,
	}

	select {
	case s.bus.BalanceResultCh <- balanceMsg:
	case <-time.After(50 * time.Millisecond):
		fmt.Printf("[gimbal_simulator] balance result channel full for %s\n", msg.CenserCode)
	}

	select {
	case s.bus.BroadcastCh <- &BroadcastMessage{
		MessageType: "balance_result",
		Data:        balanceMsg,
		Time:        msg.Time,
	}:
	case <-time.After(50 * time.Millisecond):
	}
}

func (s *GimbalSimulatorService) getOrCreateSimulator(msg *SensorRawMessage) *simulation.GimbalSimulator {
	key := msg.CenserID.String()
	sim, exists := s.simulators[key]
	if !exists {
		config := msg.Config
		if config == nil {
			config = s.createDefaultConfig()
		}
		sim = simulation.NewGimbalSimulator(config)
		s.simulators[key] = sim
	} else if msg.Config != nil {
		sim.Config = msg.Config
	}
	return sim
}

func (s *GimbalSimulatorService) createDefaultConfig() *models.SimulationConfig {
	mech := config.Mechanical
	fluid := config.Fluid

	if mech == nil || fluid == nil {
		return &models.SimulationConfig{
			InnerRingMass:        0.05,
			InnerRingRadius:      0.04,
			OuterRingMass:        0.08,
			OuterRingRadius:      0.05,
			BodyMass:             0.15,
			BodyRadius:           0.03,
			Gravity:              9.81,
			FrictionCoefficient:  0.05,
			DampingCoefficient:   0.15,
			TiltAlarmThreshold:   15.0,
			BalanceAlarmThreshold: 0.3,
			SpillAlarmThreshold:  0.5,
			PerfumeViscosity:     0.5,
			FillRatio:            0.6,
		}
	}

	return &models.SimulationConfig{
		InnerRingMass:        mech.Mechanical.InnerRing.MassKg,
		InnerRingRadius:      mech.Mechanical.InnerRing.RadiusM,
		OuterRingMass:        mech.Mechanical.OuterRing.MassKg,
		OuterRingRadius:      mech.Mechanical.OuterRing.RadiusM,
		BodyMass:             mech.Mechanical.Body.MassKg,
		BodyRadius:           mech.Mechanical.Body.RadiusM,
		Gravity:              mech.Mechanical.Environment.GravityMps2,
		FrictionCoefficient:  mech.Mechanical.Bearings.FrictionCoefficient,
		DampingCoefficient:   mech.Mechanical.Bearings.DampingCoefficient,
		TiltAlarmThreshold:   mech.AlarmThresholds.TiltAlarmDeg,
		BalanceAlarmThreshold: mech.AlarmThresholds.BalanceAlarm,
		SpillAlarmThreshold:  mech.AlarmThresholds.SpillAlarm,
		PerfumeViscosity:     fluid.GetFormula(fluid.DefaultFormula).BaseViscosityPas,
		FillRatio:            fluid.SloshDynamics.DefaultFillRatio,
	}
}

func (s *GimbalSimulatorService) SimulateResponse(censerID string, force *models.ExternalForce, duration, dt float64) ([]*models.GimbalState, []float64, error) {
	sim, exists := s.simulators[censerID]
	if !exists {
		return nil, nil, fmt.Errorf("simulator not found for censer: %s", censerID)
	}
	return simulation.SimulateGimbalResponse(sim.Config, force, duration, dt)
}

func (s *GimbalSimulatorService) CalculateBodyTilt(innerAngle, outerAngle, bodyAngle float64) float64 {
	innerRad := innerAngle * math.Pi / 180.0
	outerRad := outerAngle * math.Pi / 180.0
	bodyRad := bodyAngle * math.Pi / 180.0

	totalTiltRad := math.Acos(
		math.Cos(innerRad) * math.Cos(outerRad) * math.Cos(bodyRad),
	)

	return totalTiltRad * 180.0 / math.Pi
}
