package simulation

import (
	"math"

	"censer-simulation/models"
)

type MotionProfile struct {
	MotionType string
	Frequency  float64
	Amplitude  float64
	Duration   float64
}

var PresetMotions = map[string]MotionProfile{
	"walking": {
		MotionType: "步行",
		Frequency:  2.0,
		Amplitude:  0.5,
		Duration:   10.0,
	},
	"horse_riding": {
		MotionType: "骑马",
		Frequency:  4.0,
		Amplitude:  2.0,
		Duration:   10.0,
	},
	"car_ride": {
		MotionType: "乘车",
		Frequency:  8.0,
		Amplitude:  1.0,
		Duration:   10.0,
	},
	"running": {
		MotionType: "奔跑",
		Frequency:  6.0,
		Amplitude:  1.5,
		Duration:   10.0,
	},
	"sedan_chair": {
		MotionType: "抬轿",
		Frequency:  1.5,
		Amplitude:  0.8,
		Duration:   10.0,
	},
}

type SloshAnalyzer struct {
	Config *models.SimulationConfig
}

func NewSloshAnalyzer(config *models.SimulationConfig) *SloshAnalyzer {
	return &SloshAnalyzer{Config: config}
}

func (sa *SloshAnalyzer) calculateNaturalFrequency() float64 {
	L := sa.Config.InnerRingRadius + sa.Config.BodyRadius*0.5
	g := sa.Config.Gravity
	return math.Sqrt(g / L)
}

func (sa *SloshAnalyzer) calculateDampingRatio() float64 {
	naturalFreq := sa.calculateNaturalFrequency()
	totalMass := sa.Config.InnerRingMass + sa.Config.OuterRingMass + sa.Config.BodyMass
	momentOfInertia := (2.0/5.0)*totalMass*sa.Config.BodyRadius*sa.Config.BodyRadius +
		sa.Config.InnerRingMass*sa.Config.InnerRingRadius*sa.Config.InnerRingRadius +
		sa.Config.OuterRingMass*sa.Config.OuterRingRadius*sa.Config.OuterRingRadius

	criticalDamping := 2 * totalMass * naturalFreq
	actualDamping := sa.Config.DampingCoefficient * momentOfInertia

	return actualDamping / criticalDamping
}

func (sa *SloshAnalyzer) calculateResonanceFactor(excitationFreq float64) float64 {
	naturalFreq := sa.calculateNaturalFrequency()
	dampingRatio := sa.calculateDampingRatio()

	freqRatio := excitationFreq / naturalFreq
	denominator := math.Sqrt(
		math.Pow(1-freqRatio*freqRatio, 2) + math.Pow(2*dampingRatio*freqRatio, 2),
	)

	if denominator < 0.0001 {
		return 100
	}
	return 1.0 / denominator
}

func (sa *SloshAnalyzer) generateMotionForce(motionType string, t float64) *models.ExternalForce {
	profile, ok := PresetMotions[motionType]
	if !ok {
		profile = PresetMotions["walking"]
	}

	omega := 2 * math.Pi * profile.Frequency
	amplitude := profile.Amplitude

	force := &models.ExternalForce{
		AccelerationX: amplitude * math.Sin(omega*t),
		AccelerationY: amplitude * 0.5 * math.Sin(omega*t+math.Pi/4),
		AccelerationZ: amplitude * 0.3 * math.Sin(omega*t+math.Pi/2),
		RotationRate:  0,
	}

	if motionType == "horse_riding" {
		force.AccelerationZ *= 1.5
		force.AccelerationX *= 1.2
	}
	if motionType == "running" {
		force.AccelerationY *= 1.3
		force.RotationRate = 0.5 * math.Sin(omega*2*t)
	}

	return force
}

func (sa *SloshAnalyzer) AnalyzeMotion(motionType string) *models.SloshAnalysisResult {
	profile, ok := PresetMotions[motionType]
	if !ok {
		profile = PresetMotions["walking"]
	}

	dampingRatio := sa.calculateDampingRatio()
	resonanceFactor := sa.calculateResonanceFactor(2 * math.Pi * profile.Frequency)

	sim := NewGimbalSimulator(sa.Config)
	dt := 0.01
	duration := profile.Duration

	var maxTilt float64
	timeSeries := make([]float64, 0)
	totalTilt := 0.0
	sampleCount := 0

	for t := 0.0; t < duration; t += dt {
		force := sa.generateMotionForce(motionType, t)
		sim.Step(dt, force)
		tilt := sim.CalculateBodyTilt()

		if tilt > maxTilt {
			maxTilt = tilt
		}
		totalTilt += tilt
		sampleCount++

		if sampleCount%10 == 0 {
			timeSeries = append(timeSeries, tilt)
		}
	}

	avgTilt := totalTilt / float64(sampleCount)
	spillProbability := sa.calculateSpillProbability(maxTilt, avgTilt, resonanceFactor)
	balanceEfficiency := sa.calculateBalanceEfficiency(maxTilt, avgTilt)

	return &models.SloshAnalysisResult{
		MotionType:        profile.MotionType,
		Frequency:         profile.Frequency,
		Amplitude:         profile.Amplitude,
		DampingRatio:      dampingRatio,
		ResonanceFactor:   resonanceFactor,
		MaxTiltAngle:     maxTilt,
		SpillProbability:  spillProbability,
		BalanceEfficiency: balanceEfficiency,
		TimeSeries:        timeSeries,
	}
}

func (sa *SloshAnalyzer) AnalyzeCustomMotion(frequency, amplitude, duration float64) *models.SloshAnalysisResult {
	dampingRatio := sa.calculateDampingRatio()
	resonanceFactor := sa.calculateResonanceFactor(2 * math.Pi * frequency)

	sim := NewGimbalSimulator(sa.Config)
	dt := 0.01

	var maxTilt float64
	timeSeries := make([]float64, 0)
	totalTilt := 0.0
	sampleCount := 0

	omega := 2 * math.Pi * frequency

	for t := 0.0; t < duration; t += dt {
		force := &models.ExternalForce{
			AccelerationX: amplitude * math.Sin(omega*t),
			AccelerationY: amplitude * 0.5 * math.Cos(omega*t),
			AccelerationZ: amplitude * 0.3 * math.Sin(omega*t+math.Pi/3),
		}

		sim.Step(dt, force)
		tilt := sim.CalculateBodyTilt()

		if tilt > maxTilt {
			maxTilt = tilt
		}
		totalTilt += tilt
		sampleCount++

		if sampleCount%10 == 0 {
			timeSeries = append(timeSeries, tilt)
		}
	}

	avgTilt := totalTilt / float64(sampleCount)
	spillProbability := sa.calculateSpillProbability(maxTilt, avgTilt, resonanceFactor)
	balanceEfficiency := sa.calculateBalanceEfficiency(maxTilt, avgTilt)

	return &models.SloshAnalysisResult{
		MotionType:        "自定义运动",
		Frequency:         frequency,
		Amplitude:         amplitude,
		DampingRatio:      dampingRatio,
		ResonanceFactor:   resonanceFactor,
		MaxTiltAngle:     maxTilt,
		SpillProbability:  spillProbability,
		BalanceEfficiency: balanceEfficiency,
		TimeSeries:        timeSeries,
	}
}

func (sa *SloshAnalyzer) calculateSpillProbability(maxTilt, avgTilt, resonanceFactor float64) float64 {
	tiltThreshold := sa.Config.TiltAlarmThreshold
	if tiltThreshold <= 0 {
		tiltThreshold = 15
	}

	viscosity := sa.Config.PerfumeViscosity
	if viscosity <= 0 {
		viscosity = 0.5
	}
	fillRatio := sa.Config.FillRatio
	if fillRatio <= 0 || fillRatio > 1 {
		fillRatio = 0.6
	}

	R := sa.Config.BodyRadius
	fluidDamping := 8.0 * math.Pi * viscosity * R * R * R * fillRatio
	maxFluidDamping := 8.0 * math.Pi * 10.0 * R * R * R * 1.0
	normalizedDamping := fluidDamping / maxFluidDamping
	if normalizedDamping > 1 {
		normalizedDamping = 1
	}

	viscosityReduction := math.Exp(-normalizedDamping * 2.0)
	fillFactor := 1.0 - 0.5*fillRatio + 0.3*fillRatio*fillRatio

	tiltComponent := 0.0
	if maxTilt > tiltThreshold*0.3 {
		tiltComponent = math.Pow((maxTilt-tiltThreshold*0.3)/(tiltThreshold*0.7), 1.5)
	}

	resonanceComponent := 0.0
	if resonanceFactor > 1.5 {
		resonanceComponent = math.Pow((resonanceFactor-1.5)/3.0, 1.2) * viscosityReduction
	}

	avgTiltComponent := 0.0
	if avgTilt > tiltThreshold*0.2 {
		avgTiltComponent = (avgTilt - tiltThreshold*0.2) / (tiltThreshold * 0.8)
	}

	probability := (0.40*tiltComponent + 0.30*resonanceComponent + 0.15*avgTiltComponent) * fillFactor

	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}

	return probability
}

func (sa *SloshAnalyzer) calculateBalanceEfficiency(maxTilt, avgTilt float64) float64 {
	tiltThreshold := sa.Config.TiltAlarmThreshold
	if tiltThreshold <= 0 {
		tiltThreshold = 15
	}

	maxEfficiency := 1.0
	if maxTilt > 0 {
		maxEfficiency = math.Exp(-maxTilt / (tiltThreshold * 2))
	}

	avgEfficiency := 1.0
	if avgTilt > 0 {
		avgEfficiency = math.Exp(-avgTilt / (tiltThreshold * 1.5))
	}

	efficiency := 0.6*maxEfficiency + 0.4*avgEfficiency

	if efficiency < 0 {
		efficiency = 0
	}
	if efficiency > 1 {
		efficiency = 1
	}

	return efficiency
}

func (sa *SloshAnalyzer) FrequencyResponseAnalysis(minFreq, maxFreq float64, numPoints int) ([]float64, []float64, []float64) {
	frequencies := make([]float64, numPoints)
	amplitudes := make([]float64, numPoints)
	phases := make([]float64, numPoints)

	freqStep := (maxFreq - minFreq) / float64(numPoints-1)

	for i := 0; i < numPoints; i++ {
		freq := minFreq + float64(i)*freqStep
		frequencies[i] = freq

		rf := sa.calculateResonanceFactor(2 * math.Pi * freq)
		amplitudes[i] = rf

		naturalFreq := sa.calculateNaturalFrequency() / (2 * math.Pi)
		dampingRatio := sa.calculateDampingRatio()
		freqRatio := freq / naturalFreq
		phases[i] = math.Atan2(2*dampingRatio*freqRatio, 1-freqRatio*freqRatio) * 180.0 / math.Pi
	}

	return frequencies, amplitudes, phases
}

func (sa *SloshAnalyzer) GetNaturalFrequencyInfo() map[string]float64 {
	naturalFreq := sa.calculateNaturalFrequency()
	dampingRatio := sa.calculateDampingRatio()

	return map[string]float64{
		"natural_frequency_hz": naturalFreq / (2 * math.Pi),
		"natural_frequency_rad_s": naturalFreq,
		"damping_ratio":          dampingRatio,
		"critical_damping":          2 * (sa.Config.InnerRingMass + sa.Config.OuterRingMass + sa.Config.BodyMass) * naturalFreq,
	}
}
