package simulation

import (
	"math"

	"censer-simulation/models"
)

type GimbalSimulator struct {
	Config *models.SimulationConfig
	State  *models.GimbalState
}

func NewGimbalSimulator(config *models.SimulationConfig) *GimbalSimulator {
	return &GimbalSimulator{
		Config: config,
		State: &models.GimbalState{
			InnerAngle:    0,
			OuterAngle:    0,
			BodyAngle:     0,
			InnerVelocity: 0,
			OuterVelocity: 0,
			BodyVelocity:  0,
		},
	}
}

func (g *GimbalSimulator) Step(dt float64, force *models.ExternalForce) *models.GimbalState {
	g.outerRingDynamics(dt, force)
	g.innerRingDynamics(dt, force)
	g.bodyDynamics(dt, force)

	g.enforceMechanicalLimits()
	g.applyFriction(dt)

	return g.State
}

func (g *GimbalSimulator) outerRingDynamics(dt float64, force *models.ExternalForce) {
	I_outer := g.calculateRingMomentOfInertia(g.Config.OuterRingMass, g.Config.OuterRingRadius)

	gravityTorque := g.Config.OuterRingMass * g.Config.Gravity * g.Config.OuterRingRadius *
		math.Sin(g.State.OuterAngle*math.Pi/180.0)

	accelTorque := g.Config.OuterRingMass * g.Config.OuterRingRadius *
		math.Sqrt(force.AccelerationX*force.AccelerationX+force.AccelerationY*force.AccelerationY) *
		math.Cos(g.State.OuterAngle*math.Pi/180.0)

	angularAccel := (-gravityTorque - accelTorque) / I_outer

	g.State.OuterVelocity += angularAccel * dt * (180.0 / math.Pi)
	g.State.OuterAngle += g.State.OuterVelocity * dt
}

func (g *GimbalSimulator) innerRingDynamics(dt float64, force *models.ExternalForce) {
	I_inner := g.calculateRingMomentOfInertia(g.Config.InnerRingMass, g.Config.InnerRingRadius)

	outerAngleRad := g.State.OuterAngle * math.Pi / 180.0
	innerAngleRad := g.State.InnerAngle * math.Pi / 180.0

	gravityTorque := g.Config.InnerRingMass * g.Config.Gravity * g.Config.InnerRingRadius *
		math.Sin(innerAngleRad) * math.Cos(outerAngleRad)

	accelZ := force.AccelerationZ
	accelXY := math.Sqrt(force.AccelerationX*force.AccelerationX + force.AccelerationY*force.AccelerationY)
	accelTorque := g.Config.InnerRingMass * g.Config.InnerRingRadius *
		(accelZ*math.Cos(innerAngleRad) - accelXY*math.Sin(innerAngleRad)*math.Sin(outerAngleRad))

	angularAccel := (-gravityTorque - accelTorque) / I_inner

	g.State.InnerVelocity += angularAccel * dt * (180.0 / math.Pi)
	g.State.InnerAngle += g.State.InnerVelocity * dt
}

func (g *GimbalSimulator) bodyDynamics(dt float64, force *models.ExternalForce) {
	I_body := (2.0/5.0) * g.Config.BodyMass * g.Config.BodyRadius * g.Config.BodyRadius

	innerAngleRad := g.State.InnerAngle * math.Pi / 180.0
	outerAngleRad := g.State.OuterAngle * math.Pi / 180.0
	bodyAngleRad := g.State.BodyAngle * math.Pi / 180.0

	effectiveGravity := g.Config.Gravity * math.Cos(innerAngleRad) * math.Cos(outerAngleRad)
	gravityTorque := g.Config.BodyMass * effectiveGravity * g.Config.BodyRadius * math.Sin(bodyAngleRad)

	couplingTorque := g.Config.DampingCoefficient *
		(g.State.InnerVelocity*math.Pi/180.0 + g.State.OuterVelocity*math.Pi/180.0 - g.State.BodyVelocity*math.Pi/180.0)

	angularAccel := (-gravityTorque - couplingTorque) / I_body

	g.State.BodyVelocity += angularAccel * dt * (180.0 / math.Pi)
	g.State.BodyAngle += g.State.BodyVelocity * dt
}

func (g *GimbalSimulator) calculateRingMomentOfInertia(mass, radius float64) float64 {
	return mass * radius * radius
}

func (g *GimbalSimulator) enforceMechanicalLimits() {
	if g.State.OuterAngle > 90 {
		g.State.OuterAngle = 90
		g.State.OuterVelocity = -g.State.OuterVelocity * 0.5
	}
	if g.State.OuterAngle < -90 {
		g.State.OuterAngle = -90
		g.State.OuterVelocity = -g.State.OuterVelocity * 0.5
	}

	if g.State.InnerAngle > 90 {
		g.State.InnerAngle = 90
		g.State.InnerVelocity = -g.State.InnerVelocity * 0.5
	}
	if g.State.InnerAngle < -90 {
		g.State.InnerAngle = -90
		g.State.InnerVelocity = -g.State.InnerVelocity * 0.5
	}

	if g.State.BodyAngle > 180 {
		g.State.BodyAngle -= 360
	}
	if g.State.BodyAngle < -180 {
		g.State.BodyAngle += 360
	}
}

func (g *GimbalSimulator) applyFriction(dt float64) {
	frictionFactor := math.Exp(-g.Config.FrictionCoefficient * dt)

	g.State.OuterVelocity *= frictionFactor
	g.State.InnerVelocity *= frictionFactor
	g.State.BodyVelocity *= frictionFactor

	if math.Abs(g.State.OuterVelocity) < 0.001 {
		g.State.OuterVelocity = 0
	}
	if math.Abs(g.State.InnerVelocity) < 0.001 {
		g.State.InnerVelocity = 0
	}
	if math.Abs(g.State.BodyVelocity) < 0.001 {
		g.State.BodyVelocity = 0
	}
}

func (g *GimbalSimulator) CalculateBodyTilt() float64 {
	innerRad := g.State.InnerAngle * math.Pi / 180.0
	outerRad := g.State.OuterAngle * math.Pi / 180.0
	bodyRad := g.State.BodyAngle * math.Pi / 180.0

	totalTiltRad := math.Acos(
		math.Cos(innerRad) * math.Cos(outerRad) * math.Cos(bodyRad),
	)

	return totalTiltRad * 180.0 / math.Pi
}

func (g *GimbalSimulator) CalculateBalanceScore() float64 {
	bodyTilt := g.CalculateBodyTilt()

	threshold := g.Config.TiltAlarmThreshold
	if threshold <= 0 {
		threshold = 15
	}

	tiltScore := math.Exp(-bodyTilt * bodyTilt / (2 * threshold * threshold))

	totalVelocity := math.Abs(g.State.InnerVelocity) +
		math.Abs(g.State.OuterVelocity) + math.Abs(g.State.BodyVelocity)
	maxVelocity := 60.0
	velocityScore := math.Exp(-totalVelocity / maxVelocity)

	score := 0.7*tiltScore + 0.3*velocityScore

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

func (g *GimbalSimulator) CalculateSloshAcceleration(force *models.ExternalForce) float64 {
	innerRad := g.State.InnerAngle * math.Pi / 180.0
	outerRad := g.State.OuterAngle * math.Pi / 180.0

	rotatedAccelX := force.AccelerationX*math.Cos(outerRad) -
		force.AccelerationY*math.Sin(outerRad)
	rotatedAccelY := force.AccelerationX*math.Sin(outerRad)*math.Cos(innerRad) +
		force.AccelerationY*math.Cos(outerRad)*math.Cos(innerRad) -
		force.AccelerationZ*math.Sin(innerRad)

	effectiveAccel := math.Sqrt(
		rotatedAccelX*rotatedAccelX + rotatedAccelY*rotatedAccelY,
	)

	return effectiveAccel
}

func (g *GimbalSimulator) CalculateSpillRisk() float64 {
	bodyTilt := g.CalculateBodyTilt()
	balanceScore := g.CalculateBalanceScore()

	tiltThreshold := g.Config.TiltAlarmThreshold
	if tiltThreshold <= 0 {
		tiltThreshold = 15
	}
	balanceThreshold := g.Config.BalanceAlarmThreshold
	if balanceThreshold <= 0 {
		balanceThreshold = 0.3
	}

	tiltRisk := 0.0
	if bodyTilt > tiltThreshold*0.5 {
		tiltRisk = (bodyTilt - tiltThreshold*0.5) / (tiltThreshold * 0.5)
		if tiltRisk > 1 {
			tiltRisk = 1
		}
	}

	balanceRisk := 0.0
	if balanceScore < 1.0-balanceThreshold {
		balanceRisk = (1.0 - balanceThreshold - balanceScore) / (1.0 - balanceThreshold)
		if balanceRisk > 1 {
			balanceRisk = 1
		}
	}

	risk := 0.6*tiltRisk + 0.4*balanceRisk

	if risk < 0 {
		risk = 0
	}
	if risk > 1 {
		risk = 1
	}

	return risk
}

func SimulateGimbalResponse(config *models.SimulationConfig, force *models.ExternalForce, duration float64, dt float64) ([]*models.GimbalState, []float64) {
	sim := NewGimbalSimulator(config)
	states := make([]*models.GimbalState, 0)
	tilts := make([]float64, 0)

	for t := 0.0; t < duration; t += dt {
		state := sim.Step(dt, force)
		stateCopy := *state
		states = append(states, &stateCopy)
		tilts = append(tilts, sim.CalculateBodyTilt())
	}

	return states, tilts
}
