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

type SloshAnalyzerService struct {
	bus         *MessageBus
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	analyzers   map[string]*simulation.SloshAnalyzer
}

func NewSloshAnalyzerService(bus *MessageBus) *SloshAnalyzerService {
	ctx, cancel := context.WithCancel(context.Background())
	return &SloshAnalyzerService{
		bus:       bus,
		ctx:       ctx,
		cancel:    cancel,
		analyzers: make(map[string]*simulation.SloshAnalyzer),
	}
}

func (s *SloshAnalyzerService) Start() {
	s.running = true
	go s.processLoop()
}

func (s *SloshAnalyzerService) Stop() {
	s.running = false
	s.cancel()
}

func (s *SloshAnalyzerService) processLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case balanceMsg, ok := <-s.bus.BalanceResultCh:
			if !ok {
				return
			}
			s.processBalanceResult(balanceMsg)
		}
	}
}

func (s *SloshAnalyzerService) processBalanceResult(msg *BalanceResultMessage) {
	analyzer := s.getOrCreateAnalyzer(msg.CenserID.String())

	innerVel := msg.InnerVelocity
	outerVel := msg.OuterVelocity
	bodyVel := msg.BodyVelocity

	spillRisk := s.calculateRealTimeSpillRisk(analyzer, msg)

	metrics.SetSpillRisk(msg.CenserCode, spillRisk)

	R := analyzer.Config.BodyRadius
	viscosity := analyzer.Config.PerfumeViscosity
	fillRatio := analyzer.Config.FillRatio
	fluidDamping := 8.0 * math.Pi * viscosity * R * R * R * fillRatio

	omegaBody := math.Abs(bodyVel) * math.Pi / 180.0
	omegaInner := math.Abs(innerVel) * math.Pi / 180.0
	omegaOuter := math.Abs(outerVel) * math.Pi / 180.0
	totalOmega := omegaBody + omegaInner + omegaOuter
	criticalOmega := 3.0
	sloshExcitation := totalOmega / criticalOmega
	if sloshExcitation > 1 {
		sloshExcitation = 1
	}

	fillFactor := 1.0 - 0.6*fillRatio + 0.4*fillRatio*fillRatio

	sloshMsg := &SloshResultMessage{
		Time:            msg.Time,
		CenserID:        msg.CenserID,
		CenserCode:      msg.CenserCode,
		SpillRisk:       spillRisk,
		FluidDamping:    fluidDamping,
		SloshExcitation: sloshExcitation,
		FillFactor:      fillFactor,
		BalanceScore:    msg.BalanceScore,
		BodyTilt:        msg.BodyTilt,
		SensorData:      msg.SensorData,
	}

	if msg.SensorData != nil {
		bs := msg.BalanceScore
		sr := spillRisk
		msg.SensorData.BalanceScore = &bs
		msg.SensorData.SpillRisk = &sr
		select {
		case s.bus.PersistCh <- &PersistMessage{SensorData: msg.SensorData}:
		case <-time.After(50 * time.Millisecond):
		}
	}

	select {
	case s.bus.SloshResultCh <- sloshMsg:
	case <-time.After(50 * time.Millisecond):
		fmt.Printf("[slosh_analyzer] slosh result channel full for %s\n", msg.CenserCode)
	}

	select {
	case s.bus.BroadcastCh <- &BroadcastMessage{
		MessageType: "slosh_result",
		Data:        sloshMsg,
		Time:        msg.Time,
	}:
	case <-time.After(50 * time.Millisecond):
	}

	select {
	case s.bus.BroadcastCh <- &BroadcastMessage{
		MessageType: "sensor_data",
		Data: map[string]interface{}{
			"id":                 msg.CenserID.String(),
			"censer_code":        msg.CenserCode,
			"censer_name":        msg.CenserName,
			"timestamp":          msg.Time.Format(time.RFC3339),
			"inner_ring_angle":   msg.InnerAngle,
			"outer_ring_angle":   msg.OuterAngle,
			"body_tilt":          msg.BodyTilt,
			"slosh_acceleration": msg.SloshAccel,
			"balance_score":      msg.BalanceScore,
			"spill_risk":         spillRisk,
		},
		Time: msg.Time,
	}:
	case <-time.After(50 * time.Millisecond):
	}
}

func (s *SloshAnalyzerService) calculateRealTimeSpillRisk(analyzer *simulation.SloshAnalyzer, msg *BalanceResultMessage) float64 {
	tiltThreshold := analyzer.Config.TiltAlarmThreshold
	if tiltThreshold <= 0 {
		tiltThreshold = 15
	}
	balanceThreshold := analyzer.Config.BalanceAlarmThreshold
	if balanceThreshold <= 0 {
		balanceThreshold = 0.3
	}

	viscosity := analyzer.Config.PerfumeViscosity
	if viscosity <= 0 {
		viscosity = 0.5
	}
	fillRatio := analyzer.Config.FillRatio
	if fillRatio <= 0 || fillRatio > 1 {
		fillRatio = 0.6
	}

	R := analyzer.Config.BodyRadius
	fluidDamping := 8.0 * math.Pi * viscosity * R * R * R * fillRatio
	maxFluidDamping := 8.0 * math.Pi * 10.0 * R * R * R * 1.0
	normalizedDamping := fluidDamping / maxFluidDamping
	if normalizedDamping > 1 {
		normalizedDamping = 1
	}

	omegaBody := math.Abs(msg.BodyVelocity) * math.Pi / 180.0
	omegaInner := math.Abs(msg.InnerVelocity) * math.Pi / 180.0
	omegaOuter := math.Abs(msg.OuterVelocity) * math.Pi / 180.0
	totalOmega := omegaBody + omegaInner + omegaOuter

	criticalOmega := 3.0
	sloshExcitation := totalOmega / criticalOmega
	if sloshExcitation > 1 {
		sloshExcitation = 1
	}

	fluidDampingFactor := math.Exp(-normalizedDamping * 2.5)
	fillFactor := 1.0 - 0.6*fillRatio + 0.4*fillRatio*fillRatio

	tiltRisk := 0.0
	if msg.BodyTilt > tiltThreshold*0.5 {
		tiltRisk = (msg.BodyTilt - tiltThreshold*0.5) / (tiltThreshold * 0.5)
		if tiltRisk > 1 {
			tiltRisk = 1
		}
	}

	balanceRisk := 0.0
	if msg.BalanceScore < 1.0-balanceThreshold {
		balanceRisk = (1.0 - balanceThreshold - msg.BalanceScore) / (1.0 - balanceThreshold)
		if balanceRisk > 1 {
			balanceRisk = 1
		}
	}

	sloshRisk := sloshExcitation * fluidDampingFactor * fillFactor
	if sloshRisk > 1 {
		sloshRisk = 1
	}

	risk := 0.45*tiltRisk + 0.25*balanceRisk + 0.30*sloshRisk

	if risk < 0 {
		risk = 0
	}
	if risk > 1 {
		risk = 1
	}

	return risk
}

func (s *SloshAnalyzerService) getOrCreateAnalyzer(censerID string) *simulation.SloshAnalyzer {
	analyzer, exists := s.analyzers[censerID]
	if !exists {
		cfg := s.createDefaultConfig()
		analyzer = simulation.NewSloshAnalyzer(cfg)
		s.analyzers[censerID] = analyzer
	}
	return analyzer
}

func (s *SloshAnalyzerService) createDefaultConfig() *models.SimulationConfig {
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
		InnerRingMass:         mech.Mechanical.InnerRing.MassKg,
		InnerRingRadius:       mech.Mechanical.InnerRing.RadiusM,
		OuterRingMass:         mech.Mechanical.OuterRing.MassKg,
		OuterRingRadius:       mech.Mechanical.OuterRing.RadiusM,
		BodyMass:              mech.Mechanical.Body.MassKg,
		BodyRadius:            mech.Mechanical.Body.RadiusM,
		Gravity:               mech.Mechanical.Environment.GravityMps2,
		FrictionCoefficient:   mech.Mechanical.Bearings.FrictionCoefficient,
		DampingCoefficient:    mech.Mechanical.Bearings.DampingCoefficient,
		TiltAlarmThreshold:    mech.AlarmThresholds.TiltAlarmDeg,
		BalanceAlarmThreshold: mech.AlarmThresholds.BalanceAlarm,
		SpillAlarmThreshold:   mech.AlarmThresholds.SpillAlarm,
		PerfumeViscosity:      fluid.GetFormula(fluid.DefaultFormula).BaseViscosityPas,
		FillRatio:             fluid.SloshDynamics.DefaultFillRatio,
	}
}

func (s *SloshAnalyzerService) AnalyzeMotion(censerID, motionType string) (*models.SloshAnalysisResult, error) {
	analyzer, exists := s.analyzers[censerID]
	if !exists {
		analyzer = s.getOrCreateAnalyzer(censerID)
	}
	return analyzer.AnalyzeMotion(motionType), nil
}

func (s *SloshAnalyzerService) AnalyzeCustomMotion(censerID string, frequency, amplitude, duration float64) (*models.SloshAnalysisResult, error) {
	analyzer, exists := s.analyzers[censerID]
	if !exists {
		analyzer = s.getOrCreateAnalyzer(censerID)
	}
	return analyzer.AnalyzeCustomMotion(frequency, amplitude, duration), nil
}

func (s *SloshAnalyzerService) FrequencyResponseAnalysis(censerID string, minFreq, maxFreq float64, numPoints int) ([]float64, []float64, []float64, error) {
	analyzer, exists := s.analyzers[censerID]
	if !exists {
		analyzer = s.getOrCreateAnalyzer(censerID)
	}
	freqs, amps, phases := analyzer.FrequencyResponseAnalysis(minFreq, maxFreq, numPoints)
	return freqs, amps, phases, nil
}

func (s *SloshAnalyzerService) GetNaturalFrequencyInfo(censerID string) (map[string]float64, error) {
	analyzer, exists := s.analyzers[censerID]
	if !exists {
		analyzer = s.getOrCreateAnalyzer(censerID)
	}
	return analyzer.GetNaturalFrequencyInfo(), nil
}

func (s *SloshAnalyzerService) UpdateConfig(censerID string, config *models.SimulationConfig) {
	s.analyzers[censerID] = simulation.NewSloshAnalyzer(config)
}
